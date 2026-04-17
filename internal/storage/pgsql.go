package storage

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/qesterrx/gofermart/internal/logger"
	"github.com/qesterrx/gofermart/internal/model"
	"github.com/qesterrx/gofermart/internal/status"
	"golang.org/x/crypto/bcrypt"
)

type PGSQL struct {
	log *logger.Logger
	db  *sql.DB
}

func NewStoragePGSQL(logger *logger.Logger, dbDSN string) (*PGSQL, error) {
	llog := logger.With("PGSQL")

	//Создаем подключение
	conn, err := sql.Open("pgx", dbDSN)
	if err != nil {
		llog.Error("Ошибка подключения к БД " + err.Error())
		return nil, err
	}

	//Проверка подключения
	if err := conn.Ping(); err != nil {
		llog.Error("Ошибка при проверке соединения с БД " + err.Error())
		return nil, err
	}

	//Миграции
	driver, err := postgres.WithInstance(conn, &postgres.Config{})
	if err != nil {
		llog.Error("Ошибка при создании инстанса БД для миграций" + err.Error())
		return nil, err
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		llog.Error("Ошибка при создании экземпляра миграции " + err.Error())
		return nil, err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		llog.Error("Ошибка применения миграций" + err.Error())
		return nil, err
	}

	pg := PGSQL{log: llog, db: conn}

	//Создаем администратора
	pswd, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	pg.NewUser("admin", string(pswd))

	return &pg, nil
}

func (pg *PGSQL) NewUser(login, password string) status.Status {

	args := pgx.NamedArgs{
		"LOGIN":    login,
		"PASSWORD": password,
	}

	_, err := pg.db.Exec("INSERT INTO TUSERS (DFLOGIN, DFPASSWORD) VALUES (@LOGIN, @PASSWORD)", args)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				pg.log.Info(fmt.Sprintf("NewUser: Пользователь с логином %s уже зарегистрирован", login))
				return status.StUserAlreadyExists
			}
		}

		pg.log.Error("NewUser: " + err.Error())
		return status.StGeneralError
	}

	return status.StOk
}

func (pg *PGSQL) GetUser(login string) (*model.DBUser, status.Status) {

	args := pgx.NamedArgs{
		"LOGIN": login,
	}

	user := model.DBUser{}

	err := pg.db.QueryRow("SELECT DFUSER, DFLOGIN, DFPASSWORD FROM TUSERS WHERE DFLOGIN=@LOGIN", args).
		Scan(&user.ID, &user.Login, &user.Password)

	if err != nil {
		if err == sql.ErrNoRows {
			pg.log.Info(fmt.Sprintf("GetUser: пользователь с логином %s не найден", login))
			return nil, status.StUserNotFound
		}

		pg.log.Error("GetUser: " + err.Error())
		return nil, status.StGeneralError
	}

	return &user, status.StOk
}

func (pg *PGSQL) CheckOrderExist(order int, user int) status.Status {

	args := pgx.NamedArgs{
		"ORDER": order,
	}

	var tmpUserOrder *int
	var tmpUserWithdraw *int

	err := pg.db.QueryRow("select (SELECT DFUSER FROM TORDERS WHERE DFORDER=@ORDER), (SELECT DFUSER FROM TWITHDRAWALS WHERE DFORDER=@ORDER)", args).
		Scan(&tmpUserOrder, &tmpUserWithdraw)

	if err != nil {
		pg.log.Error("CheckOrderExist: " + err.Error())
		return status.StGeneralError
	}

	if tmpUserOrder != nil {
		if *tmpUserOrder == user {
			return status.StOrderDuplicated
		} else {
			return status.StOrderAnotherUser
		}
	}

	if tmpUserWithdraw != nil {
		if *tmpUserWithdraw == user {
			return status.StOrderDuplicated
		} else {
			return status.StOrderAnotherUser
		}
	}

	return status.StOk
}

func (pg *PGSQL) NewOrder(order *model.DBOrder) status.Status {

	args := pgx.NamedArgs{
		"USER":   order.User,
		"ORDER":  order.Order,
		"STATUS": order.Status,
	}

	_, err := pg.db.Exec("INSERT INTO TORDERS (DFORDER, DFUSER, DFSTATUS) VALUES (@ORDER, @USER, @STATUS)", args)
	if err != nil {
		pg.log.Error("NewOrder: " + err.Error())
		return status.StGeneralError
	}

	return status.StOk
}

func (pg *PGSQL) UpdateOrder(order *model.DBOrder) status.Status {

	args := pgx.NamedArgs{
		"ORDER":   order.Order,
		"STATUS":  order.Status,
		"accrual": order.Accrual,
	}

	_, err := pg.db.Exec("UPDATE TORDERS SET DFSTATUS=@STATUS, DFACCRUAL=@accrual, DFUPDATED=CURRENT_TIMESTAMP WHERE DFORDER=@ORDER", args)
	if err != nil {
		pg.log.Error("UpdateOrder: " + err.Error())
		return status.StGeneralError
	}

	if order.Accrual != nil {
		pg.log.Debug(fmt.Sprintf("change order %d status %s sum %d", order.Order, order.Status, *order.Accrual))
	} else {
		pg.log.Debug(fmt.Sprintf("change order %d status %s sum nil", order.Order, order.Status))
	}
	return status.StOk
}

func (pg *PGSQL) GetOrders(user int) (*[]model.DBOrder, status.Status) {

	args := pgx.NamedArgs{
		"USER": user,
	}

	rows, err := pg.db.Query("SELECT DFORDER, DFUSER, DFSTATUS, DFACCRUAL, DFCREATED, DFUPDATED FROM TORDERS WHERE DFUSER=@USER", args)
	if err != nil {
		pg.log.Error("GetOrders: " + err.Error())
		return nil, status.StGeneralError
	}
	defer rows.Close()

	orders := []model.DBOrder{}
	for rows.Next() {
		order := model.DBOrder{}
		err := rows.Scan(&order.Order, &order.User, &order.Status, &order.Accrual, &order.Uploaded, &order.Updated)
		if err != nil {
			pg.log.Error("GetOrders rows: " + err.Error())
			return nil, status.StGeneralError
		}

		orders = append(orders, order)
	}

	if rows.Err() != nil {
		pg.log.Error("GetOrders after rows: " + rows.Err().Error())
		return nil, status.StGeneralError
	}

	return &orders, status.StOk
}

func (pg *PGSQL) GetBalance(user int) (int, int, status.Status) {

	args := pgx.NamedArgs{
		"USER": user,
	}

	var balance int
	var withdrawals int

	err := pg.db.QueryRow("select (SELECT coalesce(SUM(DFACCRUAL),0) FROM TORDERS WHERE DFUSER=@USER), (SELECT coalesce(SUM(DFWITHDRAW),0) FROM TWITHDRAWALS t WHERE DFUSER=@USER)", args).
		Scan(&balance, &withdrawals)

	if err != nil {
		pg.log.Error("GetBalance: " + err.Error())
		return 0, 0, status.StGeneralError
	}

	balance = balance - withdrawals
	if balance < 0 {
		pg.log.Error(fmt.Sprintf("GetBalance: у пользователя %d отрицательный баланс", user))
		return 0, 0, status.StGeneralError
	}

	return balance, withdrawals, status.StOk
}

func (pg *PGSQL) NewWithdraw(withdraw *model.DBWithdraw) status.Status {

	args := pgx.NamedArgs{
		"USER":  withdraw.User,
		"ORDER": withdraw.Order,
		"SUM":   withdraw.Sum,
	}

	_, err := pg.db.Exec("INSERT INTO TWITHDRAWALS (DFORDER, DFUSER, DFWITHDRAW) VALUES (@ORDER, @USER, @SUM)", args)
	if err != nil {
		pg.log.Error("NewWithdraw: " + err.Error())
		return status.StGeneralError
	}

	return status.StOk
}

func (pg *PGSQL) GetWithdrawals(user int) (*[]model.DBWithdraw, status.Status) {

	args := pgx.NamedArgs{
		"USER": user,
	}

	rows, err := pg.db.Query("SELECT DFORDER, DFUSER, DFWITHDRAW, DFCREATED FROM TWITHDRAWALS WHERE DFUSER=@USER", args)
	if err != nil {
		pg.log.Error("GetOrders: " + err.Error())
		return nil, status.StGeneralError
	}
	defer rows.Close()

	withdrawals := []model.DBWithdraw{}
	for rows.Next() {
		withdraw := model.DBWithdraw{}
		err := rows.Scan(&withdraw.Order, &withdraw.User, &withdraw.Sum, &withdraw.Uploaded)
		if err != nil {
			pg.log.Error("GetWithdrawals rows: " + err.Error())
			return nil, status.StGeneralError
		}

		withdrawals = append(withdrawals, withdraw)
	}

	if rows.Err() != nil {
		pg.log.Error("GetWithdrawals after rows: " + rows.Err().Error())
		return nil, status.StGeneralError
	}

	return &withdrawals, status.StOk

}

func (pg *PGSQL) GetOrdersWOaccrual(limit int) (*[]model.DBOrder, status.Status) {

	args := pgx.NamedArgs{
		"LIMIT":        limit,
		"STNEW":        model.OrderStNew,
		"STPROCESSING": model.OrderStProcessing,
	}

	rows, err := pg.db.Query("SELECT DFORDER, DFUSER, DFSTATUS, DFACCRUAL, DFCREATED, DFUPDATED FROM TORDERS WHERE DFSTATUS in (@STNEW,@STPROCESSING) order by dfupdated desc fetch first @LIMIT row only", args)
	if err != nil {
		pg.log.Error("GetOrdersWOaccrual: " + err.Error())
		return nil, status.StGeneralError
	}
	defer rows.Close()

	orders := []model.DBOrder{}
	for rows.Next() {
		order := model.DBOrder{}
		err := rows.Scan(&order.Order, &order.User, &order.Status, &order.Accrual, &order.Uploaded, &order.Updated)
		if err != nil {
			pg.log.Error("GetOrdersWOaccrual rows: " + err.Error())
			return nil, status.StGeneralError
		}

		orders = append(orders, order)
	}

	if rows.Err() != nil {
		pg.log.Error("GetOrdersWOaccrual after rows: " + rows.Err().Error())
		return nil, status.StGeneralError
	}

	return &orders, status.StOk
}
