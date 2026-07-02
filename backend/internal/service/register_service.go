// internal/service/register_service.go

package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"backend/internal/mail"
	model "backend/internal/models"
	"backend/internal/repository"
	"backend/pkg/hash"

	"gorm.io/gorm"
)

func RequestRegistration(
	db *gorm.DB,
	email string,
	ip string,
) error {

	_ = ip

	email = strings.TrimSpace(email)

	if email == "" {
		return errors.New("email is required")
	}

	/*
		RECAPTCHA CHECK (DISABLED FOR POSTMAN)

		isHuman := VerifyRecaptcha(token)

		err := repository.InsertRecaptchaLog(
			db,
			email,
			token,
			isHuman,
			ip,
		)

		if err != nil {
			return err
		}

		if !isHuman {
			return errors.New("captcha verification failed")
		}
	*/

	existing, err := repository.GetLatestVerificationByEmail(db, email)

	if err == nil {
		if !existing.IsVerified && time.Now().Before(existing.ExpiresAt) {
			return errors.New("verification already sent. check your email")
		}

		if !existing.IsVerified && time.Now().After(existing.ExpiresAt) {
			err = repository.DeleteVerificationByEmail(db, email)
			if err != nil {
				return err
			}
		}
	}

	num, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return err
	}

	code := fmt.Sprintf("%06d", num.Int64())
	expiry := time.Now().Add(10 * time.Minute)

	err = repository.CreateVerificationCode(
		db,
		email,
		code,
		expiry,
	)
	if err != nil {
		return err
	}

	err = mail.SendVerificationCode(email, code)
	if err != nil {
		return err
	}

	return nil
}

func VerifyRegistrationCode(
	db *gorm.DB,
	email string,
	code string,
) error {

	email = strings.TrimSpace(email)
	code = strings.TrimSpace(code)

	verification, err := repository.GetVerificationByEmail(
		db,
		email,
	)

	if err != nil {
		return errors.New("verification not found")
	}

	if verification.IsVerified {
		return errors.New("email already verified")
	}

	if time.Now().After(verification.ExpiresAt) {
		return errors.New("verification code expired")
	}

	if verification.VerificationCode != code {
		return errors.New("invalid verification code")
	}

	err = repository.MarkVerificationAsVerified(
		db,
		verification.VerificationID,
	)

	if err != nil {
		return err
	}

	return nil
}

func CompleteRegistration(
	db *gorm.DB,
	email string,
	username string,
	password string,
	confirmPassword string,
	ip string,
	userAgent string,
) error {

	if password != confirmPassword {
		return errors.New("passwords do not match")
	}

	verification, err := repository.GetVerificationByEmail(
		db,
		email,
	)

	if err != nil {
		return errors.New("verification not found")
	}

	if !verification.IsVerified {
		return errors.New("email not verified")
	}

	existingEmail, _ := repository.GetUserByIdentifier(db, email)
	if existingEmail != nil {
		return errors.New("email already in use")
	}

	existingUsername, _ := repository.GetUserByIdentifier(db, username)
	if existingUsername != nil {
		return errors.New("username already in use")
	}

	hashedPassword, err := hash.HashPassword(password)
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	if err != nil {
		return err
	}

	expirationDate := time.Now().AddDate(1, 0, 0)

	user := model.User{
		Username:       username,
		EmailAddress:   email,
		PasswordHash:   hashedPassword,
		StatusID:       3,
		RoleID:         6,
		ExpirationDate: expirationDate,
	}

	err = repository.CreateUser(db, &user)
	if err != nil {
		return err
	}

	err = repository.InsertAuditLog(

		db,
		&user.UserID,
		1,
		2,
		"User registered",
		ip,
		userAgent,
	)

	err = repository.DeleteVerificationByEmail(db, email)
	if err != nil {
		return err
	}

	return nil
}
