package core

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"dagger.io/dagger"
	"github.com/moby/buildkit/identity"
	"github.com/stretchr/testify/require"
)

func TestProjectCmd(t *testing.T) {
	t.Parallel()

	type testCase struct {
		projectPath  string
		expectedSDK  string
		expectedName string
		expectedRoot string
	}
	for _, tc := range []testCase{
		{
			projectPath:  "core/integration/testdata/projects/go/basic",
			expectedSDK:  "go",
			expectedName: "basic",
			expectedRoot: "../../../../../../",
		},
		{
			projectPath:  "core/integration/testdata/projects/go/codetoschema",
			expectedSDK:  "go",
			expectedName: "codetoschema",
			expectedRoot: "../../../../../../",
		},
		{
			projectPath:  "core/integration/testdata/projects/python/basic",
			expectedSDK:  "python",
			expectedName: "basic",
			expectedRoot: "../../../../../../",
		},
		// TODO: add ts projects once those are under testdata too
	} {
		tc := tc
		for _, testGitProject := range []bool{false, true} {
			testGitProject := testGitProject
			testName := "local project"
			if testGitProject {
				testName = "git project"
			}
			testName += "/" + tc.projectPath
			t.Run(testName, func(t *testing.T) {
				t.Parallel()
				c, ctx := connect(t)
				stderr, err := CLITestContainer(ctx, t, c).
					WithLoadedProject(tc.projectPath, testGitProject).
					CallProject().
					Stderr(ctx)
				require.NoError(t, err)
				require.Contains(t, stderr, fmt.Sprintf(`"root": %q`, tc.expectedRoot))
				require.Contains(t, stderr, fmt.Sprintf(`"name": %q`, tc.expectedName))
				require.Contains(t, stderr, fmt.Sprintf(`"sdk": %q`, tc.expectedSDK))
			})
		}
	}
}

func TestProjectCmdInit(t *testing.T) {
	t.Parallel()

	type testCase struct {
		testName             string
		projectPath          string
		sdk                  string
		name                 string
		root                 string
		expectedErrorMessage string
	}
	for _, tc := range []testCase{
		{
			testName:    "explicit project dir/go",
			projectPath: "/var/testproject/subdir",
			sdk:         "go",
			name:        identity.NewID(),
			root:        "../",
		},
		{
			testName:    "explicit project dir/python",
			projectPath: "/var/testproject/subdir",
			sdk:         "python",
			name:        identity.NewID(),
			root:        "../..",
		},
		{
			testName:    "explicit project file",
			projectPath: "/var/testproject/subdir/dagger.json",
			sdk:         "python",
			name:        identity.NewID(),
		},
		{
			testName: "implicit project",
			sdk:      "go",
			name:     identity.NewID(),
		},
		{
			testName:    "implicit project with root",
			projectPath: "/var/testproject",
			sdk:         "python",
			name:        identity.NewID(),
			root:        "..",
		},
		{
			testName:             "invalid sdk",
			projectPath:          "/var/testproject",
			sdk:                  "c++--",
			name:                 identity.NewID(),
			expectedErrorMessage: "unsupported project SDK",
		},
		{
			testName:             "error on git",
			projectPath:          "git://github.com/dagger/dagger.git",
			sdk:                  "go",
			name:                 identity.NewID(),
			expectedErrorMessage: "project init is not supported for git projects",
		},
	} {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			c, ctx := connect(t)
			ctr := CLITestContainer(ctx, t, c).
				WithProjectArg(tc.projectPath).
				WithSDKArg(tc.sdk).
				WithNameArg(tc.name).
				CallProjectInit()

			if tc.expectedErrorMessage != "" {
				_, err := ctr.Sync(ctx)
				require.ErrorContains(t, err, tc.expectedErrorMessage)
				return
			}

			expectedConfigPath := tc.projectPath
			if !strings.HasSuffix(expectedConfigPath, "dagger.json") {
				expectedConfigPath = filepath.Join(expectedConfigPath, "dagger.json")
			}
			_, err := ctr.File(expectedConfigPath).Contents(ctx)
			require.NoError(t, err)

			stderr, err := ctr.CallProject().Stderr(ctx)
			require.NoError(t, err)
			require.Contains(t, stderr, fmt.Sprintf(`"name": %q`, tc.name))
			require.Contains(t, stderr, fmt.Sprintf(`"sdk": %q`, tc.sdk))
		})
	}

	t.Run("error on existing project", func(t *testing.T) {
		t.Parallel()
		c, ctx := connect(t)
		_, err := CLITestContainer(ctx, t, c).
			WithLoadedProject("core/integration/testdata/projects/go/basic", false).
			WithSDKArg("go").
			WithNameArg("foo").
			CallProjectInit().
			Sync(ctx)
		require.ErrorContains(t, err, "project init config path already exists")
	})
}

func TestProjectCommandHierarchy(t *testing.T) {
	t.Parallel()

	for _, sdk := range []string{"go", "python"} {
		projectDir := fmt.Sprintf("core/integration/testdata/projects/%s/basic", sdk)

		t.Run(projectDir, func(t *testing.T) {
			t.Parallel()
			c, ctx := connect(t)

			stderr, err := CLITestContainer(ctx, t, c).
				WithLoadedProject(projectDir, false).
				WithTarget("level-1:level-2:level-3:foo").
				CallDo().
				Stderr(ctx)
			require.NoError(t, err)
			require.Contains(t, stderr, "hello from foo")

			stderr, err = CLITestContainer(ctx, t, c).
				WithLoadedProject(projectDir, false).
				WithTarget("level-1:level-2:level-3:bar").
				CallDo().
				Stderr(ctx)
			require.NoError(t, err)
			require.Contains(t, stderr, "hello from bar")
		})
	}
}

func TestProjectHostExport(t *testing.T) {
	t.Parallel()

	prefix := identity.NewID()

	type testCase struct {
		sdk              string
		expectedMainFile string
	}
	for _, tc := range []testCase{
		{
			sdk:              "go",
			expectedMainFile: "main.go",
		},
		{
			sdk:              "python",
			expectedMainFile: "main.py",
		},
	} {
		tc := tc
		projectDir := fmt.Sprintf("core/integration/testdata/projects/%s/basic", tc.sdk)

		for _, testGitProject := range []bool{false, true} {
			testGitProject := testGitProject
			testName := "local project"
			if testGitProject {
				testName = "git project"
			}
			testName += "/" + projectDir
			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				t.Run("file export implicit output", func(t *testing.T) {
					t.Parallel()
					c, ctx := connect(t)
					ctr, err := CLITestContainer(ctx, t, c).
						WithLoadedProject(projectDir, testGitProject).
						WithTarget("test-file").
						WithUserArg("file-prefix", prefix).
						CallDo().
						Sync(ctx)
					if testGitProject {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
						_, err := ctr.File(filepath.Join(cliContainerRepoMntPath, prefix+"foo.txt")).Contents(ctx)
						require.NoError(t, err)
					}
				})

				t.Run("dir export implicit output", func(t *testing.T) {
					t.Parallel()
					c, ctx := connect(t)
					ctr, err := CLITestContainer(ctx, t, c).
						WithLoadedProject(projectDir, testGitProject).
						WithTarget("test-dir").
						WithUserArg("dir-prefix", prefix).
						CallDo().
						Sync(ctx)
					if testGitProject {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
						_, err = ctr.File(filepath.Join(cliContainerRepoMntPath, prefix+"subdir/subbar1.txt")).Contents(ctx)
						require.NoError(t, err)
						_, err = ctr.File(filepath.Join(cliContainerRepoMntPath, prefix+"subdir/subbar2.txt")).Contents(ctx)
						require.NoError(t, err)
						_, err = ctr.File(filepath.Join(cliContainerRepoMntPath, prefix+"bar1.txt")).Contents(ctx)
						require.NoError(t, err)
						_, err = ctr.File(filepath.Join(cliContainerRepoMntPath, prefix+"bar2.txt")).Contents(ctx)
						require.NoError(t, err)
					}
				})

				t.Run("file export explicit output", func(t *testing.T) {
					t.Parallel()
					c, ctx := connect(t)

					outputPath := "/var/blahblah.txt"
					ctr, err := CLITestContainer(ctx, t, c).
						WithLoadedProject(projectDir, testGitProject).
						WithTarget("test-file").
						WithOutputArg(outputPath).
						CallDo().
						Sync(ctx)
					require.NoError(t, err)
					_, err = ctr.File(outputPath).Contents(ctx)
					require.NoError(t, err)
				})

				t.Run("file export explicit output to parent dir", func(t *testing.T) {
					t.Parallel()
					c, ctx := connect(t)

					outputDir := "/var"
					ctr, err := CLITestContainer(ctx, t, c).
						WithLoadedProject(projectDir, testGitProject).
						WithTarget("test-file").
						WithOutputArg(outputDir).
						CallDo().
						Sync(ctx)
					require.NoError(t, err)
					_, err = ctr.File(filepath.Join(outputDir, "foo.txt")).Contents(ctx)
					require.NoError(t, err)
				})

				t.Run("dir export explicit output", func(t *testing.T) {
					t.Parallel()
					c, ctx := connect(t)

					outputDir := "/var"
					ctr, err := CLITestContainer(ctx, t, c).
						WithLoadedProject(projectDir, testGitProject).
						WithTarget("test-dir").
						WithOutputArg(outputDir).
						CallDo().
						Sync(ctx)
					require.NoError(t, err)

					_, err = ctr.File(filepath.Join(outputDir, "/subdir/subbar1.txt")).Contents(ctx)
					require.NoError(t, err)
					_, err = ctr.File(filepath.Join(outputDir, "/subdir/subbar2.txt")).Contents(ctx)
					require.NoError(t, err)
					_, err = ctr.File(filepath.Join(outputDir, "/bar1.txt")).Contents(ctx)
					require.NoError(t, err)
					_, err = ctr.File(filepath.Join(outputDir, "/bar2.txt")).Contents(ctx)
					require.NoError(t, err)
				})

				t.Run("export from container host", func(t *testing.T) {
					t.Parallel()
					c, ctx := connect(t)
					outputDir := "/var"
					ctr, err := CLITestContainer(ctx, t, c).
						WithLoadedProject(projectDir, testGitProject).
						WithTarget("test-export-local-dir").
						WithOutputArg(outputDir).
						CallDo().
						Sync(ctx)
					require.NoError(t, err)
					_, err = ctr.File(filepath.Join(outputDir, tc.expectedMainFile)).Contents(ctx)
					require.NoError(t, err)
					_, err = ctr.File(filepath.Join(outputDir, "dagger.json")).Contents(ctx)
					require.NoError(t, err)
				})
			})
		}
	}
}

func TestProjectDirImported(t *testing.T) {
	t.Parallel()

	type testCase struct {
		sdk              string
		expectedMainFile string
	}
	for _, tc := range []testCase{
		{
			sdk:              "go",
			expectedMainFile: "main.go",
		},
		{
			sdk:              "python",
			expectedMainFile: "main.py",
		},
	} {
		tc := tc
		projectDir := fmt.Sprintf("core/integration/testdata/projects/%s/basic", tc.sdk)

		for _, testGitProject := range []bool{false, true} {
			testGitProject := testGitProject
			testName := "local project"
			if testGitProject {
				testName = "git project"
			}
			testName += "/" + projectDir
			t.Run(testName, func(t *testing.T) {
				t.Parallel()
				c, ctx := connect(t)
				stderr, err := CLITestContainer(ctx, t, c).
					WithLoadedProject(projectDir, testGitProject).
					WithTarget("test-imported-project-dir").
					CallDo().
					Stderr(ctx)
				require.NoError(t, err)
				require.Contains(t, stderr, "README.md")
				require.Contains(t, stderr, projectDir)
				require.Contains(t, stderr, projectDir+"/dagger.json")
				require.Contains(t, stderr, projectDir+"/"+tc.expectedMainFile)
			})
		}
	}
}

/*
	TODO:(sipsma) more test cases to add

* Lists of structs (probably works already but add test)
* Pointer and non-pointer receiver type for structs (probably works already)
* Spread across multiple files
* Go interfaces as input/output? At least need comprehensible errors if that's not allowed yet
* Generics? At least need comprehensible errors if that's not allowed yet
* Exported and unexported methods and struct fields
* Provide multiple structs that have overlapping "type trees"
* Unnamed (inlined) structs. e.g. `type Foo struct { Bar struct { Baz string } }`
* Circular types (i.e. structs that have fields that reference themselves, etc.)
*/
func TestProjectGoCodeToSchema(t *testing.T) {
	t.Parallel()
	c, ctx := connect(t)

	// manually load project TODO: maybe this test should just use `dagger do`
	dirWithGoMod, err := filepath.Abs("../../")
	require.NoError(t, err)
	configAbsPath, err := filepath.Abs("testdata/projects/go/codetoschema/dagger.json")
	require.NoError(t, err)
	configRelPath, err := filepath.Rel(dirWithGoMod, configAbsPath)
	require.NoError(t, err)
	// TODO: have to force lazy execution of project load with Name...
	_, err = c.Project().Load(
		c.Host().Directory(dirWithGoMod),
		configRelPath,
	).Name(ctx)
	require.NoError(t, err)

	res := struct {
		Test struct {
			RequiredTypes  string
			ParentResolver struct {
				SubField string
			}
		}
	}{}
	err = c.Do(ctx,
		&dagger.Request{
			Query: `query TestCodeToSchema {
					test {
						requiredTypes(
							str: "foo",
						)
						parentResolver(str: "parent") {
							subField(str: "child")
						}
					}
				}`,
		},
		&dagger.Response{Data: &res},
	)
	require.NoError(t, err)
	require.Equal(t, `foo`, res.Test.RequiredTypes)
	require.Equal(t, "parent-child", res.Test.ParentResolver.SubField)
}