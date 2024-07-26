package quazal

// Combined exception values
const (
	UnknownError                      = 0x00010001
	OperationAborted                  = 0x00010004
	AccessDenied                      = 0x00010006
	InvalidArgument                   = 0x0001000A
	Timeout                           = 0x0001000B
	InitializationFailure             = 0x0001000C
	ConnectionFailureTypeOne          = 0x00050002
	ConnectionFailureTypeThree        = 0x00030002
	AccountDisabled                   = 0x00030067
	InvalidUsername                   = 0x00030064
	NotAuthenticated                  = 0x00030002
	InvalidPassword                   = 0x00030066
	UsernameAlreadyExists             = 0x00030068
	InvalidPID                        = 0x0003006B
	ConcurrentLoginDenied             = 0x00030069
	AccountExpired                    = 0x00030068
	EncryptionFailure                 = 0x0003006A
	InvalidOperationInLiveEnvironment = 0x0003006F
	PythonException                   = 0x00040001
	TypeError                         = 0x00040002
	IndexError                        = 0x00040003
	InvalidReference                  = 0x00040004
	CallFailure                       = 0x00040005
	MemoryError                       = 0x00040006
	KeyError                          = 0x00040007
	OperationError                    = 0x00040008
	ConversionError                   = 0x00040009
	ValidationError                   = 0x0004000A
)
