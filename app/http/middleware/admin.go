package middleware

import (
	"net/http"

	httpcontract "github.com/goravel/framework/contracts/http"
	"goravel/app/facades"
	"goravel/app/models"
)

type AdminMiddleware struct{}

func Admin() *AdminMiddleware {
	return &AdminMiddleware{}
}

func (m *AdminMiddleware) Signature() string {
	return "admin"
}

func (m *AdminMiddleware) Handle(ctx httpcontract.Context) {
	userID := ctx.Value("auth_user_id")
	if userID == nil {
		ctx.Response().Json(http.StatusUnauthorized, httpcontract.Json{
			"status":  "error",
			"message": "Belum login atau sesi telah berakhir.",
		}).Abort()
		return
	}

	var user models.User
	if err := facades.Orm().Query().Find(&user, userID); err != nil || user.ID == 0 {
		ctx.Response().Json(http.StatusUnauthorized, httpcontract.Json{
			"status":  "error",
			"message": "Pengguna tidak ditemukan.",
		}).Abort()
		return
	}

	if user.Role != "admin" {
		ctx.Response().Json(http.StatusForbidden, httpcontract.Json{
			"status":  "error",
			"message": "Akses ditolak. Fitur ini khusus administrator.",
		}).Abort()
		return
	}

	ctx.Request().Next()
}
