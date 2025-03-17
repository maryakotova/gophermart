package customerrors

var ErrUsernameTaken = &MyError{Message: "логин уже занят"}
var ErrOrderLoadedByUser = &MyError{Message: "номер заказа уже был загружен этим пользователем"}
var ErrOrderLoadedByAnotherUser = &MyError{Message: "номер заказа уже был загружен другим пользователем"}
var ErrLowBalance = &MyError{Message: "на счету недостаточно средств"}

type MyError struct {
	Message string
}

func (e *MyError) Error() string {
	return e.Message
}
