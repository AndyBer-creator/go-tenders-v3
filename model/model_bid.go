package model

// Информация о предложении
type Bid struct {
	Id string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Status *BidStatus `json:"status"`

	TenderId string `json:"tenderId"`

	AuthorType *BidAuthorType `json:"authorType"`

	AuthorId string `json:"authorId"`

	Version int32 `json:"version"`
	// Серверная дата и время в момент, когда пользователь отправил предложение на создание. Передается в формате RFC3339.
	CreatedAt string `json:"createdAt"`
}

// BidAuthorType : Тип автора
type BidAuthorType string

// List of bidAuthorType
const (
	ORGANIZATION BidAuthorType = "Organization"
	USER         BidAuthorType = "User"
)

type BidsNewBody struct {
	Name string `json:"name"`

	Description string `json:"description"`

	Status *BidStatus `json:"status"`

	TenderId string `json:"tenderId"`

	OrganizationId string `json:"organizationId"`

	CreatorUsername string `json:"creatorUsername"`
}

// BidStatus : Статус предложения
type BidStatus string

// List of bidStatus
const (
	BidStatusCREATED   BidStatus = "Created"
	BidStatusPUBLISHED BidStatus = "Published"
	BidStatusCANCELED  BidStatus = "Canceled"
	BidStatusAPPROVED  BidStatus = "Approved"
	BidStatusREJECTED  BidStatus = "Rejected"
)

// Отзыв о предложении
type BidReview struct {
	Id string `json:"id"`

	Description string `json:"description"`
	// Серверная дата и время в момент, когда пользователь отправил отзыв на предложение. Передается в формате RFC3339.
	CreatedAt string `json:"createdAt"`
}

type BidIdEditBody struct {
	Name string `json:"name,omitempty"`

	Description string `json:"description,omitempty"`
}

// BidDecision : Решение по предложению
type BidDecision string

// List of bidDecision
const (
	BidDecisionAPPROVED BidDecision = "Approved"
	BidDecisionREJECTED BidDecision = "Rejected"
)
