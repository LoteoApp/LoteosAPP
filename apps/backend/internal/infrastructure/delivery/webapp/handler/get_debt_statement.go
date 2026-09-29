package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/collections"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type GetDebtStatementHandler struct {
	getDebtStatement collections.GetDebtStatement
}

func NewGetDebtStatementHandler(getDebtStatement collections.GetDebtStatement) *GetDebtStatementHandler {
	return &GetDebtStatementHandler{getDebtStatement: getDebtStatement}
}

func (handler *GetDebtStatementHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	statement, err := handler.getDebtStatement.Execute(request.Context(), collections.Actor{
		AuthProviderID: principal.Subject,
		Roles:          principal.Roles,
	}, request.PathValue("id"))
	if err != nil {
		return err
	}
	response.WriteJSON(w, http.StatusOK, statement)
	return nil
}
