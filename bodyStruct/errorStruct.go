package bodyStruct

type ErrorStruct struct {
	Code        string      `json:"code"`
	Path        string      `json:"path"`
	Uri         string      `json:"uri"`
	RequestData interface{} `json:"requestData"`
	Request     args        `json:"test"`
	RequestId   string      `json:"requestId"`
	Error       error       `json:"error"`
	Message     string      `json:"devMessage"`
	SourceFile  string      `json:"sourceFile,omitempty"`
	// TODO 시간정보?
}

type args struct {
	QueryParam interface{} `json:"queryParam"`
	Body       interface{} `json:"body"`
	Cookie     interface{} `json:"cookie"`
	Header     interface{} `json:"header"`
}
