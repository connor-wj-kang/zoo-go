package domain

import "time"

const (
	TaskBidding        = "BIDDING"
	TaskPendingPayment = "PENDING_PAYMENT"
	TaskBooked         = "BOOKED"
	TaskInService      = "IN_SERVICE"
	TaskPendingConfirm = "PENDING_CONFIRM"
	TaskCompleted      = "COMPLETED"
	TaskSettled        = "SETTLED"
	TaskRefunding      = "REFUNDING"
	TaskExpired        = "EXPIRED"
	TaskCancelled      = "CANCELLED"

	BidActive      = "ACTIVE"
	BidWon         = "WON"
	BidLost        = "LOST"
	BidInvalidated = "INVALIDATED"
	BidWithdrawn   = "WITHDRAWN"
	BidExpired     = "EXPIRED"

	OrderUnpaid         = "UNPAID"
	OrderPaid           = "PAID"
	OrderInService      = "IN_SERVICE"
	OrderPendingConfirm = "PENDING_CONFIRM"
	OrderCompleted      = "COMPLETED"
	OrderRefunding      = "REFUNDING"
	OrderCancelled      = "CANCELLED"
	OrderRefunded       = "REFUNDED"

	PaymentPreparing = "PREPARING"
	PaymentPrepared  = "PREPARED"
	PaymentPaid      = "PAID"
	PaymentPaidLate  = "PAID_LATE"
	PaymentFailed    = "FAILED"
	PaymentClosed    = "CLOSED"
	RefundRequested  = "REQUESTED"
	RefundProcessing = "PROCESSING"
	RefundSucceeded  = "SUCCESS"
	RefundFailed     = "FAILED"
	RefundRejected   = "REJECTED"

	StudentUnverified = "UNVERIFIED"
	StudentPending    = "PENDING"
	StudentVerified   = "VERIFIED"
	StudentRejected   = "REJECTED"
	StudentExpired    = "EXPIRED"

	AccountActive   = "ACTIVE"
	AccountDisabled = "DISABLED"

	ModeBid   = "bid"
	ModeFixed = "fixed"
)

type Profile struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Avatar          string  `json:"avatar"`
	PhoneMask       string  `json:"phoneMask"`
	Verified        bool    `json:"verified"`
	CompletedOrders int     `json:"completedOrders"`
	StudentVerified bool    `json:"studentVerified"`
	StudentStatus   string  `json:"studentStatus"`
	School          string  `json:"school"`
	Rating          float64 `json:"rating"`
	DoneCount       int     `json:"doneCount"`
	ServiceRadiusKM float64 `json:"serviceRadiusKm"`
}

type Pet struct {
	ID      string   `json:"id"`
	OwnerID string   `json:"ownerId"`
	Name    string   `json:"name"`
	Species string   `json:"species"`
	Breed   string   `json:"breed"`
	Age     string   `json:"age"`
	Image   string   `json:"image"`
	Note    string   `json:"note"`
	Flags   []string `json:"flags"`
}

type Student struct {
	ID                    string     `json:"id"`
	Name                  string     `json:"name"`
	Avatar                string     `json:"avatar"`
	Rating                float64    `json:"rating"`
	DoneCount             int        `json:"doneCount"`
	DistanceKM            float64    `json:"distanceKm"`
	Intro                 string     `json:"intro"`
	Verified              bool       `json:"verified"`
	Verification          string     `json:"verification"`
	VerificationStatus    string     `json:"verificationStatus"`
	VerificationNote      string     `json:"verificationNote,omitempty"`
	VerificationExpiresAt *time.Time `json:"verificationExpiresAt,omitempty"`
	School                string     `json:"school"`
	ServiceRadius         float64    `json:"serviceRadiusKm"`
	LocationEnabled       bool       `json:"locationEnabled"`
}

type StudentVerificationApplication struct {
	ID             string     `json:"id"`
	UserID         string     `json:"userId"`
	UserName       string     `json:"userName"`
	RealName       string     `json:"realName"`
	School         string     `json:"school"`
	StudentNo      string     `json:"studentNo"`
	Major          string     `json:"major"`
	EnrollmentYear int        `json:"enrollmentYear"`
	ChsiCode       string     `json:"chsiCode"`
	Status         string     `json:"status"`
	ReviewNote     string     `json:"reviewNote,omitempty"`
	SubmittedAt    time.Time  `json:"submittedAt"`
	ReviewedAt     *time.Time `json:"reviewedAt,omitempty"`
	ValidUntil     *time.Time `json:"validUntil,omitempty"`
}

type AdminOverview struct {
	Users                int   `json:"users"`
	DisabledUsers        int   `json:"disabledUsers"`
	VerifiedStudents     int   `json:"verifiedStudents"`
	PendingVerifications int   `json:"pendingVerifications"`
	PendingRefunds       int   `json:"pendingRefunds"`
	ActiveOrders         int   `json:"activeOrders"`
	CompletedOrders      int   `json:"completedOrders"`
	TodayOrders          int   `json:"todayOrders"`
	GrossPaidFen         int64 `json:"grossPaidFen"`
}

type AdminUser struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Avatar          string    `json:"avatar"`
	PhoneMask       string    `json:"phoneMask"`
	OwnerVerified   bool      `json:"ownerVerified"`
	StudentVerified bool      `json:"studentVerified"`
	StudentStatus   string    `json:"studentStatus"`
	School          string    `json:"school"`
	AccountStatus   string    `json:"accountStatus"`
	DisabledReason  string    `json:"disabledReason,omitempty"`
	Rating          float64   `json:"rating"`
	DoneCount       int       `json:"doneCount"`
	ActiveOrders    int       `json:"activeOrders"`
	CreatedAt       time.Time `json:"createdAt"`
}

type AdminOrder struct {
	ID            string    `json:"id"`
	TaskID        string    `json:"taskId"`
	Status        string    `json:"status"`
	TaskStatus    string    `json:"taskStatus"`
	OwnerID       string    `json:"ownerId"`
	OwnerName     string    `json:"ownerName"`
	StudentID     string    `json:"studentId"`
	StudentName   string    `json:"studentName"`
	PetName       string    `json:"petName"`
	ServiceDate   string    `json:"serviceDate"`
	TimeWindow    string    `json:"timeWindow"`
	Community     string    `json:"community"`
	AmountFen     int64     `json:"amountFen"`
	FeeFen        int64     `json:"feeFen"`
	StudentFen    int64     `json:"studentFen"`
	PaymentStatus string    `json:"paymentStatus,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type AdminAuditLog struct {
	ID           string    `json:"id"`
	Actor        string    `json:"actor"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resourceType"`
	ResourceID   string    `json:"resourceId"`
	Detail       string    `json:"detail"`
	CreatedAt    time.Time `json:"createdAt"`
}

type PetSnapshot struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Species string   `json:"species"`
	Breed   string   `json:"breed"`
	Age     string   `json:"age"`
	Image   string   `json:"image"`
	Note    string   `json:"note"`
	Flags   []string `json:"flags"`
}

type Task struct {
	ID              string      `json:"id"`
	OwnerID         string      `json:"ownerId"`
	PetID           string      `json:"petId"`
	Pet             PetSnapshot `json:"pet"`
	Mode            string      `json:"mode"`
	Status          string      `json:"status"`
	BudgetFen       int64       `json:"budgetFen"`
	PriceFen        int64       `json:"priceFen"`
	ServiceDate     string      `json:"serviceDate"`
	ServiceDateText string      `json:"serviceDateText"`
	TimeWindow      string      `json:"timeWindow"`
	StartAt         time.Time   `json:"startAt"`
	Services        []string    `json:"services"`
	Community       string      `json:"community"`
	LocationName    string      `json:"locationName"`
	Latitude        float64     `json:"-"`
	Longitude       float64     `json:"-"`
	AddressMasked   string      `json:"addressMasked"`
	AddressDetail   string      `json:"addressDetail"`
	Note            string      `json:"note"`
	Checklist       []string    `json:"checklist"`
	BidDeadlineAt   time.Time   `json:"bidDeadlineAt"`
	PayDeadlineAt   *time.Time  `json:"payDeadlineAt,omitempty"`
	WinnerBidID     string      `json:"winnerBidId,omitempty"`
	OrderID         string      `json:"orderId,omitempty"`
	Photos          []string    `json:"photos"`
	Rated           bool        `json:"rated"`
	Version         int         `json:"version"`
	CreatedAt       time.Time   `json:"createdAt"`
}

type Bid struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"taskId"`
	StudentID string    `json:"studentId"`
	PriceFen  int64     `json:"priceFen"`
	Message   string    `json:"message"`
	Revision  int       `json:"revision"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Order struct {
	ID            string     `json:"id"`
	TaskID        string     `json:"taskId"`
	BidID         string     `json:"bidId"`
	OwnerID       string     `json:"ownerId"`
	StudentID     string     `json:"studentId"`
	AmountFen     int64      `json:"amountFen"`
	FeeFen        int64      `json:"feeFen"`
	StudentFen    int64      `json:"studentFen"`
	Status        string     `json:"status"`
	PaymentKey    string     `json:"paymentKey,omitempty"`
	PaymentStatus string     `json:"paymentStatus,omitempty"`
	OutTradeNo    string     `json:"outTradeNo,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	PaidAt        *time.Time `json:"paidAt,omitempty"`
	CheckInAt     *time.Time `json:"checkInAt,omitempty"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
	ConfirmedAt   *time.Time `json:"confirmedAt,omitempty"`
	RatingStars   int        `json:"ratingStars,omitempty"`
}

type BidView struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"taskId"`
	Student   Student   `json:"student"`
	PriceFen  int64     `json:"priceFen"`
	Message   string    `json:"message"`
	Revision  int       `json:"revision"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type TaskView struct {
	ID              string      `json:"id"`
	OwnerID         string      `json:"ownerId"`
	Pet             PetSnapshot `json:"pet"`
	Mode            string      `json:"mode"`
	Status          string      `json:"status"`
	BudgetFen       int64       `json:"budgetFen"`
	PriceFen        int64       `json:"priceFen"`
	ServiceDate     string      `json:"serviceDate"`
	ServiceDateText string      `json:"serviceDateText"`
	TimeWindow      string      `json:"timeWindow"`
	StartAt         time.Time   `json:"startAt"`
	Services        []string    `json:"services"`
	Community       string      `json:"community"`
	LocationName    string      `json:"locationName,omitempty"`
	DistanceKM      float64     `json:"distanceKm"`
	AddressMasked   string      `json:"addressMasked"`
	AddressDetail   string      `json:"addressDetail,omitempty"`
	Note            string      `json:"note"`
	Checklist       []string    `json:"checklist"`
	BidDeadlineAt   time.Time   `json:"bidDeadlineAt"`
	PayDeadlineAt   *time.Time  `json:"payDeadlineAt,omitempty"`
	RemainingSec    int64       `json:"remainingSec"`
	PayRemainingSec int64       `json:"payRemainingSec"`
	WinnerBidID     string      `json:"winnerBidId,omitempty"`
	OrderID         string      `json:"orderId,omitempty"`
	Photos          []string    `json:"photos"`
	Rated           bool        `json:"rated"`
	BidCount        int         `json:"bidCount"`
	Bids            []BidView   `json:"bids,omitempty"`
	MyBid           *BidView    `json:"myBid,omitempty"`
	Winner          *Student    `json:"winner,omitempty"`
	Order           *Order      `json:"order,omitempty"`
	Refund          *Refund     `json:"refund,omitempty"`
	Version         int         `json:"version"`
	CreatedAt       time.Time   `json:"createdAt"`
}

type PaymentAttempt struct {
	ID          string    `json:"id"`
	OrderID     string    `json:"orderId"`
	OutTradeNo  string    `json:"outTradeNo"`
	Description string    `json:"description"`
	PayerOpenID string    `json:"-"`
	AmountFen   int64     `json:"amountFen"`
	ExpireAt    time.Time `json:"expireAt"`
}

type PaymentRef struct {
	ID         string `json:"id"`
	OrderID    string `json:"orderId"`
	OutTradeNo string `json:"outTradeNo"`
	Status     string `json:"status"`
}

type PaymentParams struct {
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

type PaymentChallenge struct {
	PaymentID  string        `json:"paymentId"`
	OrderID    string        `json:"orderId"`
	OutTradeNo string        `json:"outTradeNo"`
	Params     PaymentParams `json:"params"`
}

type PaymentNotice struct {
	OutTradeNo    string
	TransactionID string
	TradeState    string
	AppID         string
	MchID         string
	AmountFen     int64
	SuccessAt     time.Time
}

type Refund struct {
	ID             string     `json:"id"`
	OrderID        string     `json:"orderId"`
	PaymentID      string     `json:"paymentId"`
	OutRefundNo    string     `json:"outRefundNo"`
	WeChatRefundID string     `json:"wechatRefundId,omitempty"`
	AmountFen      int64      `json:"amountFen"`
	TotalFen       int64      `json:"totalFen"`
	Reason         string     `json:"reason"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"createdAt"`
	SucceededAt    *time.Time `json:"succeededAt,omitempty"`
}

type RefundNotice struct {
	OutRefundNo    string
	WeChatRefundID string
	Status         string
	AmountFen      int64
	SuccessAt      time.Time
}

type OwnerHome struct {
	Profile    Profile          `json:"profile"`
	Area       Area             `json:"area"`
	Tasks      []TaskView       `json:"tasks"`
	Pets       []Pet            `json:"pets"`
	Address    *OwnerAddress    `json:"address,omitempty"`
	Steps      []Step           `json:"steps"`
	Guarantees []Guarantee      `json:"guarantees"`
	Community  CommunitySummary `json:"community"`
}

type OwnerAddress struct {
	Community    string  `json:"community"`
	Detail       string  `json:"detail"`
	LocationName string  `json:"locationName,omitempty"`
	Latitude     float64 `json:"latitude,omitempty"`
	Longitude    float64 `json:"longitude,omitempty"`
}

type CommunitySummary struct {
	Total  int             `json:"total"`
	Latest []CommunityPost `json:"latest"`
}

type CommunityAuthor struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Avatar          string `json:"avatar"`
	OwnerVerified   bool   `json:"ownerVerified"`
	StudentVerified bool   `json:"studentVerified"`
}

type CommunityPost struct {
	ID        string          `json:"id"`
	Author    CommunityAuthor `json:"author"`
	Content   string          `json:"content"`
	Images    []string        `json:"images"`
	CreatedAt time.Time       `json:"createdAt"`
	Mine      bool            `json:"mine"`
}

type CommunityFeed struct {
	Items      []CommunityPost `json:"items"`
	NextCursor string          `json:"nextCursor,omitempty"`
	HasMore    bool            `json:"hasMore"`
}

type Area struct {
	Name     string `json:"name"`
	District string `json:"district"`
	Enabled  bool   `json:"enabled"`
}

type Step struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type Guarantee struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type Meta struct {
	ServiceTypes         []string         `json:"serviceTypes"`
	TimeWindows          []string         `json:"timeWindows"`
	Communities          []string         `json:"communities"`
	Days                 []DateOption     `json:"days"`
	Deadlines            []DeadlineOption `json:"deadlines"`
	FeeRateBasisPoints   int64            `json:"feeRateBasisPoints"`
	BudgetMinFen         int64            `json:"budgetMinFen"`
	BudgetMaxFen         int64            `json:"budgetMaxFen"`
	PaymentTimeoutMinute int              `json:"paymentTimeoutMinutes"`
}

type DateOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type DeadlineOption struct {
	Label   string `json:"label"`
	Minutes int    `json:"minutes"`
}

type Earnings struct {
	StudentID       string          `json:"studentId"`
	TotalFen        int64           `json:"totalFen"`
	PendingFen      int64           `json:"pendingFen"`
	AvailableFen    int64           `json:"availableFen"`
	CompletedOrders int             `json:"completedOrders"`
	Records         []EarningRecord `json:"records"`
}

type EarningRecord struct {
	OrderID     string     `json:"orderId"`
	TaskID      string     `json:"taskId"`
	PetName     string     `json:"petName"`
	AmountFen   int64      `json:"amountFen"`
	FeeFen      int64      `json:"feeFen"`
	StudentFen  int64      `json:"studentFen"`
	Status      string     `json:"status"`
	ServiceDate string     `json:"serviceDate"`
	TimeWindow  string     `json:"timeWindow"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type PublishTaskRequest struct {
	OwnerID           string   `json:"ownerId"`
	PetID             string   `json:"petId"`
	Services          []string `json:"services"`
	ServiceDate       string   `json:"serviceDate"`
	ServiceDateText   string   `json:"serviceDateText"`
	TimeWindow        string   `json:"timeWindow"`
	Mode              string   `json:"mode"`
	BudgetFen         int64    `json:"budgetFen"`
	BidDurationMin    int      `json:"bidDurationMinutes"`
	Community         string   `json:"community"`
	LocationName      string   `json:"locationName"`
	Latitude          float64  `json:"latitude"`
	Longitude         float64  `json:"longitude"`
	AddressDetail     string   `json:"addressDetail"`
	Note              string   `json:"note"`
	SafetyConfirmed   bool     `json:"safetyConfirmed"`
	AgreementAccepted bool     `json:"agreementAccepted"`
}

type CreatePetRequest struct {
	Name    string   `json:"name"`
	Species string   `json:"species"`
	Breed   string   `json:"breed"`
	Age     string   `json:"age"`
	Image   string   `json:"image"`
	Note    string   `json:"note"`
	Flags   []string `json:"flags"`
}

type CreatePostRequest struct {
	Content string   `json:"content"`
	Images  []string `json:"images"`
}

type LoginRequest struct {
	Code     string `json:"code"`
	DeviceID string `json:"deviceId"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type UpdateProfileRequest struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	RadiusKM  float64 `json:"radiusKm"`
}

type StudentLocation struct {
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	RadiusKM        float64   `json:"radiusKm"`
	UpdatedAt       time.Time `json:"updatedAt"`
	LocationEnabled bool      `json:"locationEnabled"`
}

type SubmitStudentVerificationRequest struct {
	RealName       string `json:"realName"`
	School         string `json:"school"`
	StudentNo      string `json:"studentNo"`
	Major          string `json:"major"`
	EnrollmentYear int    `json:"enrollmentYear"`
	ChsiCode       string `json:"chsiCode"`
}

type ApproveStudentVerificationRequest struct {
	ValidUntil string `json:"validUntil"`
}

type RejectStudentVerificationRequest struct {
	Reason string `json:"reason"`
}

type UpdateUserStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type RejectRefundRequest struct {
	Reason string `json:"reason"`
}

type Session struct {
	Token     string    `json:"token"`
	Profile   Profile   `json:"profile"`
	ExpiresAt time.Time `json:"expiresAt"`
	IsNewUser bool      `json:"isNewUser"`
}

type SubmitBidRequest struct {
	StudentID string `json:"studentId"`
	PriceFen  int64  `json:"priceFen"`
	Message   string `json:"message"`
}

type AwardRequest struct {
	OwnerID string `json:"ownerId"`
	BidID   string `json:"bidId"`
}

type RefundRequest struct {
	OwnerID string `json:"ownerId"`
	Reason  string `json:"reason"`
}

type CompleteRequest struct {
	StudentID string   `json:"studentId"`
	Photos    []string `json:"photos"`
}

type RateRequest struct {
	OwnerID string `json:"ownerId"`
	Stars   int    `json:"stars"`
	Comment string `json:"comment"`
}
