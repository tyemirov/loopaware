package httpapi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testFaviconSiteID = "favicon-site-id"
)

func TestSiteFaviconManagerEnqueueTaskSends(testingT *testing.T) {
	manager := &SiteFaviconManager{
		workQueue: make(chan fetchTask, 1),
	}

	manager.enqueueTask(fetchTask{siteID: testFaviconSiteID})

	select {
	case enqueuedTask := <-manager.workQueue:
		require.Equal(testingT, testFaviconSiteID, enqueuedTask.siteID)
	case <-time.After(time.Second):
		testingT.Fatalf("expected task to be enqueued")
	}
}

func TestSiteFaviconManagerEnqueueTaskTimesOut(testingT *testing.T) {
	manager := &SiteFaviconManager{
		workQueue: make(chan fetchTask),
	}
	manager.inFlight.Store(testFaviconSiteID, struct{}{})

	startTime := time.Now()
	manager.enqueueTask(fetchTask{siteID: testFaviconSiteID})
	elapsedDuration := time.Since(startTime)

	_, stillInFlight := manager.inFlight.Load(testFaviconSiteID)
	require.False(testingT, stillInFlight)
	require.GreaterOrEqual(testingT, elapsedDuration, time.Second)
}
