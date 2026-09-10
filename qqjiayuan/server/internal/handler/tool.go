package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"
	"gorm.io/gorm"

	"qqjiayuan/server/pkg/resp"
)

// ToolHandler 便民服务（复刻诺哈三代 wap/tool：客户信息/ＩＰ查询/天气预报/手机归属/英汉互译/姓名评分/编码转换）
type ToolHandler struct {
	DB *gorm.DB
}

var httpClient = &http.Client{Timeout: 8 * time.Second}

// httpGetText GET 请求并返回 UTF-8 文本（自动按 GBK 解码）
func httpGetText(url string, gbk bool) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; QQjiayuan/1.0)")
	r, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if gbk {
		body, err = simplifiedchinese.GBK.NewDecoder().Bytes(body)
		if err != nil {
			return "", err
		}
	}
	return string(body), nil
}

// 客户信息（复刻 wap/tool/client.asp：手机号/访问IP/来源/浏览工具/接受文档）
func (h *ToolHandler) Client(c *gin.Context) {
	resp.OK(c, gin.H{
		"ip":      c.ClientIP(),
		"referer": c.GetHeader("Referer"),
		"ua":      c.GetHeader("User-Agent"),
		"accept":  c.GetHeader("Accept"),
	})
}

// ＩＰ查询（复刻 wap/tool/ip/index.asp：IP 或域名查归属地）
func (h *ToolHandler) IP(c *gin.Context) {
	ip := strings.TrimSpace(c.Query("ip"))
	if ip == "" {
		ip = c.ClientIP()
	}
	// 域名先解析成 IP（原版 tool.114la.com 同时支持 IP/域名）
	ipv4Re := regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`)
	if !ipv4Re.MatchString(ip) {
		addrs, err := net.LookupHost(ip)
		if err != nil || len(addrs) == 0 {
			resp.ParamError(c, "无法解析该域名，请检查输入")
			return
		}
		ip = addrs[0]
	}
	body, err := httpGetText("http://ip-api.com/json/"+ip+"?lang=zh-CN&fields=status,country,regionName,city,isp", false)
	if err != nil {
		resp.ParamError(c, "系统繁忙，请稍候再试。")
		return
	}
	var out struct {
		Status     string `json:"status"`
		Country    string `json:"country"`
		RegionName string `json:"regionName"`
		City       string `json:"city"`
		Isp        string `json:"isp"`
	}
	if json.Unmarshal([]byte(body), &out) != nil || out.Status != "success" {
		resp.ParamError(c, "系统繁忙，请稍候再试。")
		return
	}
	location := strings.Join([]string{out.Country, out.RegionName, out.City}, " ")
	resp.OK(c, gin.H{"ip": ip, "location": location, "isp": out.Isp})
}

// wmoCodeText WMO 天气代码转中文描述
func wmoCodeText(code int) string {
	m := map[int]string{
		0: "晴", 1: "基本晴", 2: "局部多云", 3: "阴", 45: "雾", 48: "雾凇",
		51: "小毛毛雨", 53: "毛毛雨", 55: "大毛毛雨", 56: "冻毛毛雨", 57: "强冻毛毛雨",
		61: "小雨", 63: "中雨", 65: "大雨", 66: "冻雨", 67: "强冻雨",
		71: "小雪", 73: "中雪", 75: "大雪", 77: "雪粒",
		80: "小阵雨", 81: "阵雨", 82: "强阵雨", 85: "小阵雪", 86: "大阵雪",
		95: "雷阵雨", 96: "雷阵雨伴冰雹", 99: "强雷阵雨伴冰雹",
	}
	if t, ok := m[code]; ok {
		return t
	}
	return "未知"
}

// 天气预报（复刻 wap/tool/weather：城市名查询，开放气象数据）
func (h *ToolHandler) Weather(c *gin.Context) {
	city := strings.TrimSpace(c.Query("city"))
	if city == "" {
		resp.ParamError(c, "请输入城市名或区号")
		return
	}
	geoBody, err := httpGetText("https://geocoding-api.open-meteo.com/v1/search?name="+urlEscape(city)+"&count=1&language=zh&format=json", false)
	if err != nil {
		resp.ParamError(c, "系统繁忙，请稍候再试。")
		return
	}
	var geo struct {
		Results []struct {
			Name      string  `json:"name"`
			Country   string  `json:"country"`
			Admin1    string  `json:"admin1"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"results"`
	}
	if json.Unmarshal([]byte(geoBody), &geo) != nil || len(geo.Results) == 0 {
		resp.ParamError(c, "没有找到该城市，请换个名字试试")
		return
	}
	g := geo.Results[0]
	fcBody, err := httpGetText(fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current=temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m&daily=weather_code,temperature_2m_max,temperature_2m_min&timezone=auto&forecast_days=3",
		g.Latitude, g.Longitude), false)
	if err != nil {
		resp.ParamError(c, "系统繁忙，请稍候再试。")
		return
	}
	var fc struct {
		Current struct {
			Temperature float64 `json:"temperature_2m"`
			Humidity    int     `json:"relative_humidity_2m"`
			WeatherCode int     `json:"weather_code"`
			WindSpeed   float64 `json:"wind_speed_10m"`
		} `json:"current"`
		Daily struct {
			Time   []string  `json:"time"`
			Code   []int     `json:"weather_code"`
			Max    []float64 `json:"temperature_2m_max"`
			Min    []float64 `json:"temperature_2m_min"`
		} `json:"daily"`
	}
	if json.Unmarshal([]byte(fcBody), &fc) != nil {
		resp.ParamError(c, "系统繁忙，请稍候再试。")
		return
	}
	place := g.Name
	if g.Admin1 != "" && g.Admin1 != g.Name {
		place = g.Admin1 + "·" + g.Name
	}
	daily := []gin.H{}
	for i := range fc.Daily.Time {
		daily = append(daily, gin.H{"date": fc.Daily.Time[i], "weather": wmoCodeText(fc.Daily.Code[i]),
			"max": fc.Daily.Max[i], "min": fc.Daily.Min[i]})
	}
	resp.OK(c, gin.H{"city": place, "current": gin.H{
		"temp": fc.Current.Temperature, "weather": wmoCodeText(fc.Current.WeatherCode),
		"humidity": fc.Current.Humidity, "wind": fc.Current.WindSpeed}, "daily": daily})
}

// carrierOf 号段前3位 → 运营商（本地静态表）
func carrierOf(prefix string) string {
	m := map[string]string{}
	for _, p := range []string{"134", "135", "136", "137", "138", "139", "147", "148", "150", "151", "152", "157", "158", "159", "178", "182", "183", "184", "187", "188", "195", "197", "198"} {
		m[p] = "移动"
	}
	for _, p := range []string{"130", "131", "132", "145", "146", "155", "156", "166", "167", "171", "175", "176", "185", "186", "196"} {
		m[p] = "联通"
	}
	for _, p := range []string{"133", "149", "153", "162", "173", "174", "177", "180", "181", "189", "190", "191", "193", "199"} {
		m[p] = "电信"
	}
	for _, p := range []string{"170"} {
		m[p] = "虚拟运营商"
	}
	if v, ok := m[prefix]; ok {
		return v
	}
	return ""
}

// 手机归属地（复刻 wap/tool/phone：原版调用114la已失效；本地识别运营商+尝试外部查省市）
func (h *ToolHandler) Phone(c *gin.Context) {
	phone := strings.TrimSpace(c.Query("phone"))
	if !regexp.MustCompile(`^1[0-9]{10}$`).MatchString(phone) {
		resp.ParamError(c, "输入错误！")
		return
	}
	corp := carrierOf(phone[:3])
	province, city := "", ""
	// 淘宝号段接口（可用则补充省市/区号）
	if body, err := httpGetText("https://tcc.taobao.com/cc/json/mobile_tel_segment.htm?tel="+phone, true); err == nil && strings.Contains(body, "__GetZoneResult_") {
		get := func(key string) string {
			re := regexp.MustCompile(key + `\s*[:=]\s*'([^']*)'`)
			m := re.FindStringSubmatch(body)
			if len(m) < 2 {
				return ""
			}
			return m[1]
		}
		province = get("province")
		city = get("city")
		if cat := get("catName"); cat != "" {
			corp = cat
		}
	}
	data := gin.H{"phone": phone, "corp": corp}
	if province != "" {
		data["province"] = province
		data["city"] = city
	} else {
		data["note"] = "归属地查询服务暂不可用，已识别运营商"
	}
	resp.OK(c, data)
}

// 英汉互译（复刻 wap/tool/translate：中英互译）
func (h *ToolHandler) Translate(c *gin.Context) {
	text := strings.TrimSpace(c.Query("text"))
	if text == "" {
		resp.ParamError(c, "请输入要翻译的内容")
		return
	}
	// 含中文 → 译英；否则 → 译中（对齐原版"英汉互译"双向）
	pair := "en|zh-CN"
	hasCJK := false
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			hasCJK = true
			break
		}
	}
	if hasCJK {
		pair = "zh-CN|en"
	}
	body, err := httpGetText("https://api.mymemory.translated.net/get?q="+urlEscape(text)+"&langpair="+pair, false)
	if err != nil {
		resp.ParamError(c, "系统繁忙，请稍候再试。")
		return
	}
	var out struct {
		ResponseData struct {
			TranslatedText string `json:"translatedText"`
		} `json:"responseData"`
		ResponseStatus int `json:"responseStatus"`
	}
	if json.Unmarshal([]byte(body), &out) != nil || out.ResponseStatus != 200 || out.ResponseData.TranslatedText == "" {
		resp.ParamError(c, "系统繁忙，请稍候再试。")
		return
	}
	resp.OK(c, gin.H{"text": text, "translated": out.ResponseData.TranslatedText})
}

// 姓名评分（复刻 wap/tool/name：原版调用搜搜打分接口，已失效；本地娱乐算法替代）
func (h *ToolHandler) Name(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" || utf8.RuneCountInString(name) > 8 {
		resp.ParamError(c, "请输入正确的姓名")
		return
	}
	sum := 0
	for _, r := range name {
		sum += int(r)
	}
	score := 61 + sum%39 // 61~99
	persona := []string{"聪慧睿智，气质独特，遇事有主见", "温润如玉，待人真诚，人缘极佳", "敢想敢做，意志坚定，天生领袖气质", "心思细腻，富有艺术细胞，想象力丰富"}
	career := []string{"事业运极佳，中年前后可成大器", "稳中有升，宜文教、技术方向发展", "贵人运旺，合作共赢成就一番事业", "早年辛劳，厚积薄发，后福无穷"}
	love := []string{"情缘深厚，易遇良人白首偕老", "桃花不断，需擦亮双眼择佳偶", "感情专一，家庭和睦幸福", "聚少离多，宜多沟通经营"}
	i := sum % 4
	resp.OK(c, gin.H{"name": name, "score": score,
		"persona": persona[i], "career": career[(sum/3)%4], "love": love[(sum/7)%4]})
}

// urlEscape 简单 URL 编码
func urlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' || r == '~' {
			b.WriteRune(r)
		} else {
			for _, by := range []byte(string(r)) {
				fmt.Fprintf(&b, "%%%02X", by)
			}
		}
	}
	return b.String()
}
