package dailywriter

import (
	"fmt"
	"sort"
	"testing"
	"time"
)

func TestSort(tt *testing.T) {
	base := time.Now()
	timeCnt := []int64{1, 2, 10, 4, 8}

	var c = []logInfo{}
	for _, t := range timeCnt {
		c = append(c, logInfo{base.Add(time.Duration(t) * time.Second), nil})
	}
	sort.Sort(byFormatTime(c))
	sort.Slice(timeCnt, func(i, j int) bool {
		return timeCnt[i] >= timeCnt[j]
	})
	for index, t := range timeCnt {
		if !c[index].timestamp.Equal(base.Add(time.Duration(t) * time.Second)) {
			tt.Error("error with sort")
		}
	}

	defaultFilenameDateFormat = "2006-01-02-15-04"
	tttt,_:=time.ParseInLocation(defaultFilenameDateFormat,"2019-07-12-19-19",time.Local)
	fmt.Println(tttt,time.Now().Local())

}
