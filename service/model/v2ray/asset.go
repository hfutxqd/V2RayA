package v2ray

import (
	"V2RayA/global"
	"V2RayA/tools"
	"errors"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

var v2rayLocationAsset *string

func GetV2rayLocationAsset() (s string) {
	if v2rayLocationAsset != nil {
		return *v2rayLocationAsset
	}
	switch global.ServiceControlMode {
	case global.DockerMode:
		return "/etc/v2ray"
	case global.SystemctlMode, global.ServiceMode:
		p, _ := GetV2rayServiceFilePath()
		out, err := exec.Command("sh", "-c", "cat "+p+"|grep Environment=V2RAY_LOCATION_ASSET").CombinedOutput()
		if err != nil {
			break
		}
		s = strings.TrimSpace(string(out))
		s = s[len("Environment=V2RAY_LOCATION_ASSET="):]
	}
	var err error
	if s == "" {
		//默认为v2ray运行目录
		s, err = GetV2rayWorkingDir()
	}
	if err != nil {
		//再不行只能盲猜一个
		s = "/etc/v2ray"
	}
	v2rayLocationAsset = &s
	return
}

func GetV2rayWorkingDir() (string, error) {
	switch global.ServiceControlMode {
	case global.SystemctlMode, global.ServiceMode:
		//从systemd的启动参数里找
		p, _ := GetV2rayServiceFilePath()
		out, err := exec.Command("sh", "-c", "cat "+p+"|grep ExecStart=").CombinedOutput()
		if err != nil {
			return "", errors.New(err.Error() + string(out))
		}
		arr := strings.Split(strings.TrimSpace(string(out)), " ")
		return path.Dir(arr[0][len("ExecStart="):]), nil
	case global.CommonMode:
		//从环境变量里找
		out, err := exec.Command("sh", "-c", "which v2ray").CombinedOutput()
		if err == nil {
			return path.Dir(strings.TrimSpace(string(out))), nil
		}
	case global.DockerMode:
		//只能指望在asset里有没有了
		asset := GetV2rayLocationAsset()
		_, err := os.Stat(asset + "/v2ray")
		if err != nil {
			return "", err
		}
		return asset, nil
	}
	return "", errors.New("not found")
}

func GetV2ctlDir() (string, error) {
	d, err := GetV2rayWorkingDir()
	if err == nil {
		_, err := os.Stat(d + "/v2ctl")
		if err != nil {
			return "", err
		}
		return d, nil
	}
	out, err := exec.Command("sh", "-c", "which v2ctl").Output()
	if err != nil {
		err = errors.New(err.Error() + string(out))
		return "", err
	}
	return path.Dir(strings.TrimSpace(string(out))), nil
}

func IsH2yExists() bool {
	_, err := os.Stat(GetV2rayLocationAsset() + "/h2y.dat")
	if err != nil {
		return false
	}
	return true
}
func IsGeoipExists() bool {
	_, err := os.Stat(GetV2rayLocationAsset() + "/geoip.dat")
	if err != nil {
		return false
	}
	return true
}
func IsGeositeExists() bool {
	_, err := os.Stat(GetV2rayLocationAsset() + "/geosite.dat")
	if err != nil {
		return false
	}
	return true
}
func GetH2yModTime() (time.Time, error) {
	return tools.GetFileModTime(GetV2rayLocationAsset() + "/h2y.dat")
}
func IsCustomExists() bool {
	_, err := os.Stat(GetV2rayLocationAsset() + "/custom.dat")
	if err != nil {
		return false
	}
	return true
}
func GetCustomModTime() (time.Time, error) {
	return tools.GetFileModTime(GetV2rayLocationAsset() + "/custom.dat")
}

func GetConfigPath() (path string) {
	switch global.ServiceControlMode {
	case global.CommonMode:
		path = GetV2rayLocationAsset() + "/config.json"
	default:
		path = "/etc/v2ray/config.json"
	}
	return
}
