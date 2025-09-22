package api

import (
	"go-tenders-v3/utils"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type Handlers struct {
	Bid     *BidHandler
	Tender  *TenderHandler
	Service *ServiceHandler
}

type Route struct {
	Name    string
	Method  string
	Pattern string
	//HandlerFunc http.HandlerFunc
	HandlerName string
}

type Routes []Route

func NewRouter(handlers *Handlers) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	// Создаем мапу строк-обработчиков
	handlerMapping := map[string]http.HandlerFunc{
		"Index":              handlers.Service.Index,
		"CheckServer":        handlers.Service.CheckServer,
		"CreateBid":          handlers.Bid.CreateBid,
		"EditBid":            handlers.Bid.EditBid,
		"GetBidReviews":      handlers.Bid.GetBidReviews,
		"GetBidStatus":       handlers.Bid.GetBidStatus,
		"GetBidsForTender":   handlers.Bid.GetBidsForTender,
		"GetUserBids":        handlers.Bid.GetUserBids,
		"RollbackBid":        handlers.Bid.RollbackBid,
		"SubmitBidDecision":  handlers.Bid.SubmitBidDecision,
		"SubmitBidFeedback":  handlers.Bid.SubmitBidFeedback,
		"UpdateBidStatus":    handlers.Bid.UpdateBidStatus,
		"CreateTender":       handlers.Tender.CreateTender,
		"EditTender":         handlers.Tender.EditTender,
		"GetTenderStatus":    handlers.Tender.GetTenderStatus,
		"GetTenders":         handlers.Tender.GetTenders,
		"GetUserTenders":     handlers.Tender.GetUserTenders,
		"RollbackTender":     handlers.Tender.RollbackTender,
		"UpdateTenderStatus": handlers.Tender.UpdateTenderStatus,

		// Добавьте сюда остальные обработчики
	}

	for _, route := range routes {
		var handler http.Handler

		if route.HandlerName != "" {
			h, ok := handlerMapping[route.HandlerName]
			if !ok {
				handler = http.NotFoundHandler()
			} else {
				handler = h
			}
		} else {
			handler = http.NotFoundHandler()
		}

		handler = utils.Logger(handler, route.Name)

		router.Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}

var routes = Routes{
	{
		Name:        "Index",
		Method:      "GET",
		Pattern:     "/api/",
		HandlerName: "Index",
	},

	{
		Name:        "CheckServer",
		Method:      strings.ToUpper("Get"),
		Pattern:     "/api/ping",
		HandlerName: "CheckServer",
	},

	{
		Name:        "CreateBid",
		Method:      strings.ToUpper("Post"),
		Pattern:     "/api/bids/new",
		HandlerName: "CreateBid",
	},

	{
		Name:        "CreateTender",
		Method:      strings.ToUpper("Post"),
		Pattern:     "/api/tenders/new",
		HandlerName: "CreateTender",
	},

	{
		Name:        "EditBid",
		Method:      strings.ToUpper("Patch"),
		Pattern:     "/api/bids/{bidId}/edit",
		HandlerName: "EditBid",
	},

	{
		Name:        "EditTender",
		Method:      strings.ToUpper("Patch"),
		Pattern:     "/api/tenders/{tenderId}/edit",
		HandlerName: "EditTender",
	},

	{
		Name:        "GetBidReviews",
		Method:      strings.ToUpper("Get"),
		Pattern:     "/api/bids/{tenderId}/reviews",
		HandlerName: "GetBidReviews",
	},

	{
		Name:        "GetBidStatus",
		Method:      strings.ToUpper("Get"),
		Pattern:     "/api/bids/{bidId}/status",
		HandlerName: "GetBidStatus",
	},

	{
		Name:        "GetBidsForTender",
		Method:      strings.ToUpper("Get"),
		Pattern:     "/api/bids/{tenderId}/list",
		HandlerName: "GetBidsForTender",
	},

	{
		Name:        "GetTenderStatus",
		Method:      strings.ToUpper("Get"),
		Pattern:     "/api/tenders/{tenderId}/status",
		HandlerName: "GetTenderStatus",
	},

	{
		Name:        "GetTenders",
		Method:      strings.ToUpper("Get"),
		Pattern:     "/api/tenders",
		HandlerName: "GetTenders",
	},

	{
		Name:        "GetUserBids",
		Method:      strings.ToUpper("Get"),
		Pattern:     "/api/bids/my",
		HandlerName: "GetUserBids",
	},

	{
		Name:        "GetUserTenders",
		Method:      strings.ToUpper("Get"),
		Pattern:     "/api/tenders/my",
		HandlerName: "GetUserTenders",
	},

	{
		Name:        "RollbackBid",
		Method:      strings.ToUpper("Put"),
		Pattern:     "/api/bids/{bidId}/rollback/{version}",
		HandlerName: "RollbackBid",
	},

	{
		Name:        "RollbackTender",
		Method:      strings.ToUpper("Put"),
		Pattern:     "/api/tenders/{tenderId}/rollback/{version}",
		HandlerName: "RollbackTender",
	},

	{
		Name:        "SubmitBidDecision",
		Method:      strings.ToUpper("Put"),
		Pattern:     "/api/bids/{bidId}/submit_decision",
		HandlerName: "SubmitBidDecision",
	},

	{
		Name:        "SubmitBidFeedback",
		Method:      strings.ToUpper("Put"),
		Pattern:     "/api/bids/{bidId}/feedback",
		HandlerName: "SubmitBidFeedback",
	},

	{
		Name:        "UpdateBidStatus",
		Method:      strings.ToUpper("Put"),
		Pattern:     "/api/bids/{bidId}/status",
		HandlerName: "UpdateBidStatus",
	},

	{
		Name:        "UpdateTenderStatus",
		Method:      strings.ToUpper("Put"),
		Pattern:     "/api/tenders/{tenderId}/status",
		HandlerName: "UpdateTenderStatus",
	},
}
