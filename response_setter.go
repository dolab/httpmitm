package httpmitm

type ResponseSetter interface {
	ResponseSet(mark, data string, force bool)
}
