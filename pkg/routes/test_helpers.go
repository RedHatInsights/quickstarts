package routes

// Common test response types
type ResponseBody struct {
	Id   uint   `json:"id"`
	Name string `json:"name"`
}

type ResponsePayload struct {
	Data []ResponseBody
}

type SingleResponsePayload struct {
	Data ResponseBody
}

type MessageResponsePayload struct {
	Msg string `json:"msg"`
}
