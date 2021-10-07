package utils

// TODO get the ip
func GetPublicIp( ) string {
	return "0.0.0.0"
}

func CheckFreeTrial( plan string ) bool {
	if plan == "expired" {
		return true
	}
	return false
}