package repository

import "embed"

//go:embed sql/*.sql
var sqlFS embed.FS

func mustSQL(path string) string {
	b, err := sqlFS.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(b)
}

var (
	createUserQuery              = mustSQL("sql/createuser.sql")
	getUserByLoginQuery          = mustSQL("sql/getuserbylogin.sql")
	createOrderQuery             = mustSQL("sql/createorder.sql")
	getOrderByNumberQuery        = mustSQL("sql/getorderbynumber.sql")
	listOrdersByUserQuery        = mustSQL("sql/listordersbyuser.sql")
	listOrdersForProcessingQuery = mustSQL("sql/listordersforprocessing.sql")
	getBalanceByUserQuery        = mustSQL("sql/getbalancebyuser.sql")
	listWithdrawalsByUserQuery   = mustSQL("sql/listwithdrawalsbyuser.sql")
)
