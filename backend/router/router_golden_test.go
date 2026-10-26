package router_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
	"medtrust-backend/router"
)

func TestGoldenRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := router.SetupRouter()

	routes := r.Routes()
	var routeStrings []string
	for _, route := range routes {
		routeStrings = append(routeStrings, fmt.Sprintf("%-7s %s", route.Method, route.Path))
	}
	sort.Strings(routeStrings)

	goldenPath := filepath.Join("..", "..", "tests", "golden", "routes.txt")
	content := ""
	for _, s := range routeStrings {
		content += s + "\n"
	}

	// 如果金样文件不存在，则首次创建
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		err := os.WriteFile(goldenPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("写入路由金样文件失败: %v", err)
		}
		t.Logf("✅ 首次创建路由金样文件，共 %d 条路由", len(routeStrings))
		return
	}

	// 比对金样
	existing, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("读取路由金样文件失败: %v", err)
	}

	if string(existing) != content {
		t.Fatalf("❌ 路由表与金样不一致！\n期望:\n%s\n实际:\n%s", string(existing), content)
	}

	t.Logf("✅ 路由表金样比对通过，共 %d 条路由完全一致", len(routeStrings))
}
