package httpapi_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	widgetJavaScriptAssetRelativePath                 = "assets/widget.js"
	documentMutationObserverAttributeFilterExpression = "attributeFilter: [\"class\", \"data-theme\", \"data-bs-theme\", \"style\"]"
	bodyMutationObserverAttributeFilterExpression     = "attributeFilter: [\"class\", \"data-bs-theme\", \"style\"]"
)

func TestWidgetJavaScriptIncludesBootstrapThemeObservers(testContext *testing.T) {
	_, testFilePath, _, callerResolved := runtime.Caller(0)
	require.True(testContext, callerResolved)
	widgetJavaScriptAssetPath := filepath.Join(filepath.Dir(testFilePath), widgetJavaScriptAssetRelativePath)

	widgetJavaScriptContent, readWidgetAssetError := os.ReadFile(widgetJavaScriptAssetPath)
	require.NoError(testContext, readWidgetAssetError)

	testCases := []struct {
		testCaseName      string
		requiredSubstring string
	}{
		{
			testCaseName:      "DocumentMutationObserverIncludesBootstrapThemeAttribute",
			requiredSubstring: documentMutationObserverAttributeFilterExpression,
		},
		{
			testCaseName:      "BodyMutationObserverIncludesBootstrapThemeAttribute",
			requiredSubstring: bodyMutationObserverAttributeFilterExpression,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		testContext.Run(testCase.testCaseName, func(testContext *testing.T) {
			testContext.Helper()
			require.Contains(testContext, string(widgetJavaScriptContent), testCase.requiredSubstring)
		})
	}
}
