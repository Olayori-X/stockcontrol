package sqltools

import (
	"time"

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
	AddUser(user models.User) (string, error)
	SetupDatabase() error
	GetUsers() ([]models.User, error)
	UpsertLoggedInUser(userID string, code string, role string) error
	UpdateUserCode(userID string, hashedCode string) error
	UserLoggedIn(userid string) *AuthenticatedUser
	UpdateUserProfile(user models.User) error
	AddForgotPasswordRecord(userID, code string) error
	ChangeUserPassword(email string, hashedPassword string) error
	CreatePickupRequest(req *models.PickupRequest) error
	ConfirmPickupRequest(requestID, distributorID string) (bool, error)
	SearchDistributors(query string, excludeID string) ([]models.User, error)
	GetPendingPickupRequests(distributorID string) ([]models.PendingPickupRequest, error)
	GetUnacceptedPickupRequests(salesAssociateID string) ([]models.PendingPickupRequest, error)
	AddProduct(product models.Products) error
	GetProducts() ([]models.Products, error)
	GetProductBySKU(sku string) (*models.Products, error)
	EditProduct(sku string, updated models.Products) error
	DeleteProduct(sku string) error
	SearchUsers(query string, excludeID string) ([]models.User, error)
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
