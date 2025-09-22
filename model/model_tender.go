package model

// Информация о тендере
type Tender struct {
	Id string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	ServiceType *TenderServiceType `json:"serviceType"`

	Status *TenderStatus `json:"status"`

	OrganizationId string `json:"organizationId"`

	Version int32 `json:"version"`
	// Серверная дата и время в момент, когда пользователь отправил тендер на создание. Передается в формате RFC3339.
	CreatedAt string `json:"createdAt"`
}

type TendersNewBody struct {
	Name string `json:"name"`

	Description string `json:"description"`

	ServiceType *TenderServiceType `json:"serviceType"`

	Status *TenderStatus `json:"status"`

	OrganizationId string `json:"organizationId"`

	CreatorUsername string `json:"creatorUsername"`
}

// TenderStatus : Статус тендер
type TenderStatus string

// List of tenderStatus
const (
	CREATED   TenderStatus = "Created"
	PUBLISHED TenderStatus = "Published"
	CLOSED    TenderStatus = "Closed"
)

// TenderServiceType : Вид услуги, к которой относиться тендер
type TenderServiceType string

// List of tenderServiceType
const (
	CONSTRUCTION TenderServiceType = "Construction"
	DELIVERY     TenderServiceType = "Delivery"
	MANUFACTURE  TenderServiceType = "Manufacture"
)

type TenderIdEditBody struct {
	Name string `json:"name,omitempty"`

	Description string `json:"description,omitempty"`

	ServiceType *TenderServiceType `json:"serviceType,omitempty"`
}
