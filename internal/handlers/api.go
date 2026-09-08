package handlers

import (
	admin "github.com/Olayori-X/stock-control-backend/internal/handlers/admin"
	outlet "github.com/Olayori-X/stock-control-backend/internal/handlers/admin/outlets"
	product "github.com/Olayori-X/stock-control-backend/internal/handlers/admin/products"
	route "github.com/Olayori-X/stock-control-backend/internal/handlers/admin/routes"
	auth "github.com/Olayori-X/stock-control-backend/internal/handlers/auth"
	general "github.com/Olayori-X/stock-control-backend/internal/handlers/general"
	pickup "github.com/Olayori-X/stock-control-backend/internal/handlers/pickup"
	middleware "github.com/Olayori-X/stock-control-backend/internal/middleware"
	chimiddle "github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

func Handler(r *chi.Mux) {
	r.Use(chimiddle.StripSlashes)

	r.Route("/auth", func(router chi.Router) {
		router.Post("/login", auth.LoginHandler)
		router.Post("/signup", auth.SignupHandler)
	})

	r.Route("/admin", func(router chi.Router) {
		// Middle ware for /account authorization
		router.Use(middleware.Authorization)
		router.Use(middleware.RequireRole("admin"))
		router.Post("/signup", auth.SignupHandler)
		router.Post("/addproduct", product.AddProductHandler)
		router.Get("/products", product.GetProductsHandler)
		router.Get("/productbysku", product.GetProductBySKUHandler)
		router.Put("/editproduct", product.EditProductHandler)
		router.Delete("/deleteproduct", product.DeleteProductHandler)
		router.Get("/search", general.SearchUsersHandler)
		router.Post("/changepassword", auth.ChangePasswordHandler)
		router.Get("/users", admin.GetUsersHandler)
		router.Post("/routeplan", route.SetRoutePlanHandler)
		router.Get("/routeplan", route.GetRoutePlanHandler)
		router.Post("/routeplan/approve", route.ApproveRoutePlanHandler)
		router.Post("/addoutlet", outlet.AddOutletHandler)
		router.Get("/outlets", outlet.GetOutletsHandler)
		router.Get("/outletbyid", outlet.GetOutletByIDHandler)
		router.Put("/editoutlet", outlet.EditOutletHandler)
		router.Delete("/deactivateoutlet", outlet.SetOutletActiveHandler)
	})

	r.Route("/sales", func(router chi.Router) {
		router.Use(middleware.Authorization)
		router.Use(middleware.RequireRole("sales"))
		router.Post("/createrequest", pickup.CreatePickupRequestHandler)
		router.Get("/searchdistributor", pickup.SearchDistributorsHandler)
		router.Get("/unacceptedrequests", pickup.GetUnacceptedPickupRequestsHandler)
		router.Get("/products", product.GetProductsHandler)
		router.Post("/confirmvisit", outlet.ConfirmOutletVisitHandler)
		router.Post("/submitsale", outlet.SubmitSaleHandler)
	})

	r.Route("/distributor", func(router chi.Router) {
		router.Use(middleware.Authorization)
		router.Use(middleware.RequireRole("distributor"))
		router.Get("/pendingrequests", pickup.GetPendingPickupRequestsHandler)
		router.Post("/confirmrequest", pickup.ConfirmPickupRequestHandler)
	})
}
