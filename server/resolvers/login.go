package resolvers

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
	"golang.org/x/crypto/bcrypt"
)

func Login(ctx context.Context, params model.LoginInput) (*model.AuthResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.AuthResponse
	if err != nil {
		return res, err
	}

	if constants.DISABLE_BASIC_AUTHENTICATION == "true" {
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}

	params.Email = strings.ToLower(params.Email)
	user, err := db.Mgr.GetUserByEmail(params.Email)
	if err != nil {
		return res, fmt.Errorf(`user with this email not found`)
	}

	if !strings.Contains(user.SignupMethod, enum.BasicAuth.String()) {
		return res, fmt.Errorf(`user has not signed up email & password`)
	}

	if user.EmailVerifiedAt <= 0 {
		return res, fmt.Errorf(`email not verified`)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password))

	if err != nil {
		log.Println("Compare password error:", err)
		return res, fmt.Errorf(`invalid password`)
	}
	userIdStr := fmt.Sprintf("%d", user.ID)
	refreshToken, _, _ := utils.CreateAuthToken(utils.UserAuthInfo{
		ID:    userIdStr,
		Email: user.Email,
	}, enum.RefreshToken)

	accessToken, expiresAt, _ := utils.CreateAuthToken(utils.UserAuthInfo{
		ID:    userIdStr,
		Email: user.Email,
	}, enum.AccessToken)

	session.SetToken(userIdStr, refreshToken)

	res = &model.AuthResponse{
		Message:              `Logged in successfully`,
		AccessToken:          &accessToken,
		AccessTokenExpiresAt: &expiresAt,
		User: &model.User{
			ID:              userIdStr,
			Email:           user.Email,
			Image:           &user.Image,
			FirstName:       &user.FirstName,
			LastName:        &user.LastName,
			SignupMethod:    user.SignupMethod,
			EmailVerifiedAt: &user.EmailVerifiedAt,
			CreatedAt:       &user.CreatedAt,
			UpdatedAt:       &user.UpdatedAt,
		},
	}

	utils.SetCookie(gc, accessToken)

	return res, nil
}