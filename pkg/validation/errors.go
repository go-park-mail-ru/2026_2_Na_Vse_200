package validation

import (
	"errors"
	"fmt"
)

// Причины отказа. Текст попадает пользователю под поле формы как есть,
// поэтому он с заглавной буквы: обычное правило Go про строчные ошибки
// рассчитано на склейку при оборачивании, а эти ошибки не оборачиваются.
var (
	ErrEmailRequired = errors.New("Укажите email")
	ErrEmailInvalid  = errors.New("Некорректный email")

	ErrPasswordRequired = errors.New("Укажите пароль")
	ErrPasswordShort    = fmt.Errorf("Пароль должен быть не короче %d символов", _passwordMinRunes)
	ErrPasswordLong     = errors.New("Пароль слишком длинный")
	ErrPasswordSimple   = errors.New("Пароль должен содержать хотя бы одну букву и одну цифру")

	ErrDisplayNameRequired = errors.New("Укажите имя")
	ErrDisplayNameLength   = fmt.Errorf("Имя должно содержать от %d до %d символов",
		_displayNameMinLen, _displayNameMaxLen)
)
