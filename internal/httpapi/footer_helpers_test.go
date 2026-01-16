package httpapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFooterConfigForVariantRejectsUnknown(testingT *testing.T) {
	_, configErr := footerConfigForVariant(footerVariant("unknown"))
	require.Error(testingT, configErr)
}

func TestRenderFooterHTMLForVariantRejectsUnknown(testingT *testing.T) {
	_, renderErr := renderFooterHTMLForVariant(footerVariant("unknown"))
	require.Error(testingT, renderErr)
}
