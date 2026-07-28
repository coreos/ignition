// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.)

package util

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	expectedBadDirModes = []int{
		500,  // 0764
		550,  // 01046
		555,  // 01053
		700,  // 01274
		750,  // 01356
		755,  // 01363
		770,  // 01402
		775,  // 01407
		777,  // 01411
		1700, // 03244
		1750, // 03326
		1755, // 03333
		1770, // 03352
		1775, // 03357
		1777, // 03361
		2700, // 05214
		2750, // 05276
		2755, // 05303
		2770, // 05322
		2775, // 05327
		2777, // 05331
		3700, // 07164
		3750, // 07246
		3755, // 07253
		3770, // 07272
		3775, // 07277
		3777, // 07301
	}
	expectedBadFileModes = []int{
		400,  // 0620
		440,  // 0670
		444,  // 0674
		500,  // 0764
		550,  // 01046
		555,  // 01053
		600,  // 01130
		640,  // 01200
		644,  // 01204
		660,  // 01224
		664,  // 01230
		666,  // 01232
		700,  // 01274
		750,  // 01356
		755,  // 01363
		770,  // 01402
		775,  // 01407
		777,  // 01411
		2500, // 04704
		2550, // 04766
		2555, // 04773
		2700, // 05214
		2750, // 05276
		2755, // 05303
		2770, // 05322
		2775, // 05327
		4500, // 010624
		4550, // 010706
		4555, // 010713
		4700, // 011134
		4750, // 011216
		4755, // 011223
		4770, // 011242
		4775, // 011247
		6500, // 014544
		6550, // 014626
		6555, // 014633
		6700, // 015054
		6750, // 015136
		6755, // 015143
		6770, // 015162
		6775, // 015167
	}
)

func TestCheckForDecimalMode(t *testing.T) {
	// test decimal to octal conversion
	for i := -1; i < 10001; i++ {
		t.Run(fmt.Sprintf("mode %d", i), func(t *testing.T) {
			iStr := fmt.Sprintf("%d", i)
			result, ok := decimalModeToOctal(i)

			assert.Equal(t, i >= 0 && i <= 7777 && !strings.ContainsAny(iStr, "89"), ok, "converting to octal returned incorrect ok")
			if ok {
				assert.Equal(t, iStr, fmt.Sprintf("%o", result), "converting to octal failed")
			}
		})
	}

	// check the checker against a hardcoded list
	var badDirModes []int
	var badFileModes []int
	for i := -1; i <= 10000; i++ {
		if CheckForDecimalMode(i, true) != nil {
			badDirModes = append(badDirModes, i)
		}
		if CheckForDecimalMode(i, false) != nil {
			badFileModes = append(badFileModes, i)
		}
	}
	assert.Equal(t, expectedBadDirModes, badDirModes, "bad set of decimal directory modes")
	assert.Equal(t, expectedBadFileModes, badFileModes, "bad set of decimal file modes")
}
