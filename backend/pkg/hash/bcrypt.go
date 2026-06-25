package hash

import "golang.org/x/crypto/bcrypt"

func CheckPassword(hashedPassword string, plainPassword string) error {
	return bcrypt.CompareHashAndPassword(
		[]byte(hashedPassword),
		[]byte(plainPassword),
	)
}
