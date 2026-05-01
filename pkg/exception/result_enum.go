package exception

const (
	// AuthenticationTokenInvalid /** token非法 */
	AuthenticationTokenInvalid = -1001
	// AuthenticationTokenExpired /** token过期 */
	AuthenticationTokenExpired = -1002
	// NotInTest /** 在正式环境尝试访问测试接口 */
	NotInTest = -1003
	// WechatServerError /** 微信服务端的错误 */
	WechatServerError = -1100
	// UserAlreadyExists /** 用户已存在 */
	UserAlreadyExists = 10001
	// UserOrPasswordWrong /** 用户账号或密码错误 */
	UserOrPasswordWrong = 10002
	// InvalidSmsVerifyCode /** 手机验证码错误 */
	InvalidSmsVerifyCode = 10003
	// GuestBoundAnotherAccount /** 游客用户已经绑定其他账户，请用其他账户登陆 */
	GuestBoundAnotherAccount = 10004
	// SourceAndPlatformIdRegistered /** 如果指定的来源和第三方账号已经注册过了，就不能使用，因为这样会造成两个账号的冲突。*/
	SourceAndPlatformIdRegistered = 10005
	// ServiceNotGrantedForApp /** 指定服务没有授权给指定App使用 */
	ServiceNotGrantedForApp = 10006
	// MissingParameter /** 缺少参数 */
	MissingParameter = 10007
	// NotBindPhone /** 未绑定手机号 */
	NotBindPhone = 10008
	// UserNotFound /** 用户未找到 */
	UserNotFound = 10009
	// PhoneNumberUsed /** 手机号被占用 */
	PhoneNumberUsed = 10010

	InternalServerErr = 500

	NotSupportedSource = 10011

	IllegalParameters = 10012

	WrongAppId = -1004
)

var resultMsg = map[int]string{
	AuthenticationTokenExpired:    "token过期",
	AuthenticationTokenInvalid:    "token非法",
	NotInTest:                     "在正式环境尝试访问测试接口",
	WechatServerError:             "微信服务端的错误",
	UserAlreadyExists:             "用户已存在",
	UserOrPasswordWrong:           "用户账号或密码错误",
	InvalidSmsVerifyCode:          "手机验证码错误",
	GuestBoundAnotherAccount:      "游客用户已经绑定其他账户，请用其他账户登陆",
	SourceAndPlatformIdRegistered: "指定的来源和第三方账号已经注册过了",
	ServiceNotGrantedForApp:       "指定服务没有授权给指定App使用",
	MissingParameter:              "缺少参数",
	NotBindPhone:                  "未绑定手机号",
	UserNotFound:                  "用户未找到",
	PhoneNumberUsed:               "手机号被占用",
	InternalServerErr:             "服务异常",
	NotSupportedSource:            "不支持的渠道",
	WrongAppId:                    "错误的AppId",
	IllegalParameters:             "错误的请求参数",
}

var resultHttpCode = map[int]int{
	AuthenticationTokenExpired:    400,
	AuthenticationTokenInvalid:    400,
	NotInTest:                     400,
	WechatServerError:             400,
	UserAlreadyExists:             400,
	UserOrPasswordWrong:           400,
	InvalidSmsVerifyCode:          400,
	GuestBoundAnotherAccount:      400,
	SourceAndPlatformIdRegistered: 400,
	ServiceNotGrantedForApp:       400,
	MissingParameter:              400,
	NotBindPhone:                  400,
	UserNotFound:                  400,
	PhoneNumberUsed:               400,
	InternalServerErr:             500,
	NotSupportedSource:            400,
	WrongAppId:                    400,
	IllegalParameters:             400,
}

// GetResultMsg 获取错误描述
func GetResultMsg(code int) string {
	return resultMsg[code]
}

// GetResultHttpCode 获取code对应HttpCode
func GetResultHttpCode(code int) int {
	return resultHttpCode[code]
}
