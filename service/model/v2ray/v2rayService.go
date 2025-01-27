package v2ray

import (
	"V2RayA/global"
	"V2RayA/tools"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

func EnableV2rayService() (err error) {
	var out []byte
	switch global.ServiceControlMode {
	case global.DockerMode, global.CommonMode: //docker, common中无需enable service
	case global.ServiceMode:
		out, err = exec.Command("sh", "-c", "update-rc.d v2ray enable").CombinedOutput()
		if err != nil {
			err = errors.New(err.Error() + string(out))
		}
	case global.SystemctlMode:
		out, err = exec.Command("sh", "-c", "systemctl enable v2ray").CombinedOutput()
		if err != nil {
			err = errors.New(err.Error() + string(out))
		}
	}
	return
}

func DisableV2rayService() (err error) {
	var out []byte
	switch global.ServiceControlMode {
	case global.DockerMode, global.CommonMode: //docker, common中无需disable service
	case global.ServiceMode:
		out, err = exec.Command("sh", "-c", "update-rc.d v2ray disable").CombinedOutput()
		if err != nil {
			err = errors.New(err.Error() + string(out))
		}
	case global.SystemctlMode:
		out, err = exec.Command("sh", "-c", "systemctl disable v2ray").CombinedOutput()
		if err != nil {
			err = errors.New(err.Error() + string(out))
		}
	}
	return
}

func GetV2rayServiceFilePath() (path string, err error) {
	var out []byte

	if global.ServiceControlMode == global.SystemctlMode {
		out, err = exec.Command("sh", "-c", "systemctl status v2ray|grep Loaded|awk '{print $3}'").Output()
		if err != nil {
			path = `/usr/lib/systemd/system/v2ray.service`
		}
	} else if global.ServiceControlMode == global.ServiceMode {
		out, err = exec.Command("sh", "-c", "service v2ray status|grep Loaded|awk '{print $3}'").Output()
		if err != nil || strings.TrimSpace(string(out)) == "(Reason:" {
			path = `/lib/systemd/system/v2ray.service`
		}
	} else {
		err = errors.New("当前环境无法使用systemctl和service命令")
		return
	}
	sout := strings.TrimSpace(string(out))
	path = sout[1 : len(sout)-1]
	return
}

func LiberalizeProcFile() (err error) {
	if global.ServiceControlMode != global.SystemctlMode && global.ServiceControlMode != global.ServiceMode {
		return
	}
	p, err := GetV2rayServiceFilePath()
	if err != nil {
		return
	}
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return
	}
	s := string(b)
	if strings.Contains(s, "LimitNPROC=500") && strings.Contains(s, "LimitNOFILE=1000000") {
		return
	}
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "LimitNPROC=") || strings.HasPrefix(lines[i], "LimitNOFILE=") {
			lines = append(lines[:i], lines[i+1:]...)
		}
	}
	for i, line := range lines {
		if strings.ToLower(line) == "[service]" {
			s = strings.Join(lines[:i+1], "\n")
			s += "\nLimitNPROC=500\nLimitNOFILE=1000000\n"
			s += strings.Join(lines[i+1:], "\n")
			break
		}
	}
	err = ioutil.WriteFile(p, []byte(s), os.ModeAppend)
	if err != nil {
		return
	}
	if IsV2RayRunning() {
		err = RestartV2rayService()
	}
	return
}

func IsV2rayServiceValid() bool {
	switch global.ServiceControlMode {
	case global.SystemctlMode:
		out, err := exec.Command("sh", "-c", "systemctl list-unit-files|grep v2ray.service").Output()
		return err == nil && len(bytes.TrimSpace(out)) > 0
	case global.ServiceMode:
		out, err := exec.Command("sh", "-c", "service v2ray status|grep not-found").Output()
		return err == nil && len(bytes.TrimSpace(out)) == 0
	case global.DockerMode:
		return IsGeoipExists() && IsGeositeExists()
	case global.CommonMode:
		if !IsGeoipExists() || !IsGeositeExists() {
			return false
		}
		out, err := exec.Command("sh", "-c", "which v2ray").Output()
		return err == nil && len(bytes.TrimSpace(out)) > 0
	}
	return false
}

func GetV2rayServiceVersion() (ver string, err error) {
	dir, err := GetV2rayWorkingDir()
	if err != nil || len(dir) <= 0 {
		return "", errors.New("无法找到v2ray可执行文件")
	}
	out, err := exec.Command("sh", "-c", fmt.Sprintf("%v/v2ray -version|awk '{print $2}'|awk 'NR==1'", dir)).Output()
	return strings.TrimSpace(string(out)), err
}

func IfTProxyModLoaded() bool {
	out, err := exec.Command("sh", "-c", "lsmod|awk '{print $1}'|grep ^xt_TPROXY$").Output()
	return err == nil && len(bytes.TrimSpace(out)) > 0
}

func CheckTProxySupported() (err error) {
	ver, err := GetV2rayServiceVersion()
	if err != nil {
		return errors.New("获取v2ray-core版本失败")
	}
	if greaterEqual, err := tools.VersionGreaterEqual(ver, "4.19.1"); err != nil || !greaterEqual {
		return errors.New("v2ray-core版本低于4.19.1")
	}
	if !IfTProxyModLoaded() && global.ServiceControlMode != global.DockerMode { //docker下无法判断
		var out []byte
		out, err = exec.Command("sh", "-c", "modprobe xt_TPROXY").CombinedOutput()
		if err != nil {
			if !strings.Contains(string(out), "not found") {
				return errors.New("启动xt_TPROXY失败: " + string(out))
			}
			// modprobe失败，不支持xt_TPROXY方案
			return errors.New("不支持xt_TPROXY" + string(out))
		}
	}
	return
}
