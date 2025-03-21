package controller

import (
	"context"
	"net/http"
	"shared/core"
	"shared/utility"
	"strings"

	"github.com/google/uuid"
)

const requestIDKey core.ContextKey = "REQUEST_ID"

func RequestIDMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

const UserIDContext core.ContextKey = "userID"

const UserAccessContext core.ContextKey = "userAccess"

func GetBearerToken(w http.ResponseWriter, r *http.Request) (string, string, bool) {

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", "authorization header required", false
	}

	bearerToken := strings.Split(authHeader, " ")
	if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
		return "", "invalid Authorization header format", false
	}

	return bearerToken[1], "", true
}

func Authentication(next http.HandlerFunc, jwt utility.JWTTokenizer) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		// bearerToken, errMessage, ok := GetBearerToken(w, r)
		// if !ok {
		// 	utility.WriteJSON(w, http.StatusUnauthorized, utility.Response{Status: "failed", Error: &errMessage})
		// 	return
		// }

		// content, err := jwt.VerifyToken(bearerToken)
		// if err != nil {
		// 	msg := "unverified token"
		// 	utility.WriteJSON(w, http.StatusUnauthorized, utility.Response{Status: "failed", Error: &msg})
		// 	return
		// }

		// var userTokenPayload model.UserTokenPayload
		// if err := json.Unmarshal(content, &userTokenPayload); err != nil {
		// 	msg := "incorrect token payload"
		// 	utility.WriteJSON(w, http.StatusUnauthorized, utility.Response{Status: "failed", Error: &msg})
		// 	return
		// }

		// ctx := core.AttachDataToContext(r.Context(), UserAccessContext, userTokenPayload.UserAccess)
		// ctx = core.AttachDataToContext(ctx, UserIDContext, userTokenPayload.UserID)

		// r = r.WithContext(ctx)

		next.ServeHTTP(w, r)

	}

}

// func Authorization(next http.HandlerFunc, access model.Access) http.HandlerFunc {
func Authorization(next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		// // by pass
		// if access == "0" {
		// 	next.ServeHTTP(w, r)
		// 	return
		// }

		// userAccess := core.GetDataFromContext(r.Context(), UserAccessContext, model.UserAccess(""))

		// if !userAccess.HasAccess(access) {
		// 	msg := "unauthorized operation"
		// 	utility.WriteJSON(w, http.StatusForbidden, utility.Response{Status: "failed", Error: &msg})
		// 	return
		// }

		next.ServeHTTP(w, r)

	}
}
