package sqltools

import (
	"time"

	"github.com/Olayori-X/stock-control-backend/api"
	"github.com/Olayori-X/stock-control-backend/models"
	log "github.com/sirupsen/logrus"
)

type LoginDetails struct {
	Password string
	UserID   string
	Role     string
	Verified bool
}

type AuthenticatedUser struct {
	Code      string
	UserID    string
	Role      string
	LoginTime time.Time
}

type CoinDetails struct {
	Coins    int64
	Username string
}

type DatabaseInterface interface {
	GetUserLoginDetails(username string) *LoginDetails
	GetUserDetails(userID string) *models.User
	AddUser(user *models.User) error
	SetupDatabase() error
	GetUsers() ([]models.User, error)
	UpsertLoggedInUser(userID string, code string, role string) error
	UpdateUserCode(userID string, hashedCode string) error
	UserLoggedIn(userid string) *AuthenticatedUser
	UpdateUserProfile(user models.User) error
	AddForgotPasswordRecord(userID, code string) error
	ChangeUserPassword(email string, hashedPassword string) error
	CreatePickupRequest(req *models.PickupRequest) error
	ConfirmPickupRequest(requestID, distributorID string) (bool, *models.Invoice, error) // signature changed
	SearchDistributors(query string) ([]models.User, error)
	GetPendingPickupRequests(distributorID string) ([]models.PendingPickupRequest, error)
	GetUnacceptedPickupRequests(salesAssociateID string) ([]models.PendingPickupRequest, error)
	AddProduct(product models.Products) error
	GetProducts() ([]models.Products, error)
	GetProductBySKU(sku string) (*models.Products, error)
	EditProduct(sku string, updated models.Products) error
	DeleteProduct(sku string) error
	SearchUsers(query string, excludeID string) ([]models.User, error)

	GetPlannedOutletsForDay(salesAssociateID, routeDay string) ([]PlannedOutletLocation, error)
	GetSettingFloat(key string) (float64, error)
	RecordResumption(salesAssociateID, routeDay string, lat, lon, distanceM float64, result, deviceRef string) error

	SetRoutePlan(salesAssociateID, routeDay string, outletIDs []string) error
	GetRoutePlan(salesAssociateID, routeDay string) (*models.RoutePlan, error)
	ApproveRoutePlan(salesAssociateID, routeDay string) (bool, error)

	AddOutlet(outlet *models.Outlet) error
	GetOutlets(includeInactive bool) ([]models.Outlet, error)
	GetOutletByID(outletID string) (*models.Outlet, error)
	EditOutlet(outlet *models.Outlet) (*models.Outlet, error)
	SetOutletActive(outletID string, active bool) (bool, error)
	RecordOutletVisit(salesAssociateID, outletID, routeDay string, lat, lon, distanceM float64, status string) error
	SubmitSale(salesAssociateID string, input *api.SubmitSaleInput) (sale *models.Sale, alreadyExisted bool, blocked bool, err error)

	RecordPayment(invoiceID string, amount float64) (*models.Receipt, *models.Invoice, error)
	GetOutstandingInvoices(distributorID string) ([]models.Invoice, error)

	SetUserPIN(userID, pinHash string) error
	GetUserPINLoginDetails(userID string) *PINLoginDetails

	AssignDistributor(salesAssociateID, distributorID string) error
	UnassignDistributor(salesAssociateID, distributorID string) (bool, error)
	GetAssignedDistributors(salesAssociateID string) ([]models.DistributorAssignment, error)
	IsDistributorAssigned(salesAssociateID, distributorID string) (bool, error)
	GetOutsideCoverage(salesAssociateID string, weekStart, weekEnd time.Time) (*models.OutsideCoverageReport, error)
}

func NewDatabase() (*DatabaseInterface, error) {
	var database DatabaseInterface = &RealDB{}
	var err error = database.SetupDatabase()

	if err != nil {
		log.Error("Failed to set up database connection: ", err)
		return nil, err
	}

	return &database, nil
}
