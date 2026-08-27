package validator

type Errors map[string]string

func New() Errors {
	return make(Errors)
}

func (e Errors) Err() error {
	if len(e) != 0 {
		return e
	}

	return nil
}

func (e Errors) Error() string {
	return "validator: input validation failed"
}

func (e Errors) Check(ok bool, key, message string) {
	if !ok {
		_, exists := e[key]
		if !exists {
			e[key] = message
		}
	}
}
