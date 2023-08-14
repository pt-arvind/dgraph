//go:build integration || upgrade

/*
 * Copyright 2023 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package bulk

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgraph/chunker"
	"github.com/dgraph-io/dgraph/dgraphtest"
)

// JSON output can be hundreds of lines and diffs can scroll off the terminal before you
// can look at them. This option allows saving the JSON to a specified directory instead
// for easier reviewing after the test completes.
//var savedir = flag.String("savedir", "",
//	"directory to save json from test failures in")
//var quiet = flag.Bool("quiet", false,
//	"just output whether json differs, not a diff")

func (bsuite *BulkTestSuite) TestQueriesFor21Million() {
	t := bsuite.T()
	_, thisFile, _, _ := runtime.Caller(0)
	queryDir := filepath.Join(filepath.Dir(thisFile), "../queries")

	files, err := os.ReadDir(queryDir)
	if err != nil {
		t.Fatalf("Error reading directory: %s", err.Error())
	}

	//savepath := ""
	//diffs := 0
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "query-") {
			continue
		}

		bsuite.Run(file.Name(), func() {
			require.NoError(t, bsuite.bulkLoader())

			// start alphas
			//require.NoError(t, c.Start())
			require.NoError(t, bsuite.StartAlpha())

			// Upgrade
			bsuite.Upgrade()

			// For this test we DON'T want to start with an empty database.
			dg, cleanup, err := bsuite.dc.Client()
			defer cleanup()
			require.NoError(t, err)

			filename := filepath.Join(queryDir, file.Name())
			reader, cleanup := chunker.FileReader(filename, nil)
			bytes, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("Error reading file: %s", err.Error())
			}
			contents := string(bytes[:])
			cleanup()

			// The test query and expected result are separated by a delimiter.
			bodies := strings.SplitN(contents, "\n---\n", 2)
			// Dgraph can get into unhealthy state sometime. So, add retry for every query.
			for retry := 0; retry < 3; retry++ {
				// If a query takes too long to run, it probably means dgraph is stuck and there's
				// no point in waiting longer or trying more tests.
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				resp, err := dg.NewTxn().Query(ctx, bodies[0])
				cancel()

				if retry < 2 && (err != nil || ctx.Err() == context.DeadlineExceeded) {
					continue
				}

				if ctx.Err() == context.DeadlineExceeded {
					t.Fatal("aborting test due to query timeout")
				}

				t.Logf("running %s", file.Name())
				//if *savedir != "" {
				//	savepath = filepath.Join(*savedir, file.Name())
				//}

				dgraphtest.CompareJSON(bodies[1], string(resp.GetJson()))
			}
		})
	}
	//
	//if *savedir != "" && diffs > 0 {
	//	t.Logf("test json saved in directory: %s", *savedir)
	//}
}
