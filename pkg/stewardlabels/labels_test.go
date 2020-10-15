package stewardlabels

import (
	"fmt"
	"testing"

	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/davecgh/go-spew/spew"
	"github.com/mohae/deepcopy"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TestPropagateSuccessTestCase struct {
	// fields must be exported to make deepcopy work

	LabelsToPropagate []string

	To       map[string]string
	From     map[string]string
	Add      map[string]string
	Expected map[string]string
}

func Test__propagate__Success(t *testing.T) {
	propagatedLabels := []string{
		"propagated1",
		"propagated2",
	}
	testcases := []TestPropagateSuccessTestCase{}

	// testcases: no change to original labels
	for _, to := range []map[string]string{
		nil,
		{},
		{"key1": "to"},
		{"key1": "to", "key2": "to"},
		{"propagated1": "propagated1Value", "propagated2": "propagated2Value"},
	} {
		for _, from := range []map[string]string{
			nil,
			{},
			{"key1": "from"},
			{"key1": "from", "key2": "from"},
		} {
			for _, add := range []map[string]string{
				nil,
				{},
				{"key1": "add"},
				{"key1": "add", "key2": "add"},
			} {
				testcases = append(testcases, TestPropagateSuccessTestCase{
					LabelsToPropagate: propagatedLabels,
					To:                to,
					From:              from,
					Add:               add,
					Expected:          to, // regardless of from or add
				})
			}
		}
	}

	// testcases: merge
	for _, key := range propagatedLabels {
		for _, to := range []map[string]string{
			nil,
			{},
			{key: "value1"},
		} {
			// add has key but not from
			for _, from := range []map[string]string{
				nil,
				{},
				{"key1": "from"},
				{key: "value1"},
			} {
				for _, add := range []map[string]string{
					nil,
					{},
					{"key1": "add"},
					{key: "value1"},
				} {
					if from[key] != "" || add[key] != "" {
						testcases = append(testcases, TestPropagateSuccessTestCase{
							LabelsToPropagate: propagatedLabels,
							To:                to,
							From:              from,
							Add:               add,
							Expected:          map[string]string{key: "value1"},
						})
					}
				}
			}
		}
	}

	// execute
	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			tc := deepcopy.Copy(tc).(TestPropagateSuccessTestCase)
			t.Parallel()
			logTestCaseSpec(t, tc)

			// SETUP
			toObj := &DummyObject1{}
			toObj.SetLabels(tc.To)

			fromObj := &DummyObject1{}
			fromObj.SetLabels(tc.From)

			// EXERCISE
			resultErr := propagate(tc.LabelsToPropagate, toObj, fromObj, tc.Add)

			// VERIFY
			assert.NilError(t, resultErr)
			assert.DeepEqual(t, tc.Expected, toObj.GetLabels())
		})
	}
}

type TestPropagateFailsTestCase struct {
	// fields must be exported To make deepcopy work

	LabelsToPropagate []string

	To   map[string]string
	From map[string]string
	Add  map[string]string

	ExpectedOrigValue      string
	ExpectedOverwriteValue string
}

func Test__propagate__FailsOnOverwrite(t *testing.T) {
	testcases := []TestPropagateFailsTestCase{}

	// generate testcases
	const propagated1 = "propagated1"

	for _, to := range []map[string]string{
		nil,
		{},
		{propagated1: "to"},
	} {
		for _, from := range []map[string]string{
			nil,
			{},
			{"key1": "from"},
			{propagated1: "from"},
		} {
			for _, add := range []map[string]string{
				nil,
				{},
				{"key1": "add"},
				{propagated1: "add"},
			} {
				concurrentValues := []string{}
				if to[propagated1] != "" {
					concurrentValues = append(concurrentValues, to[propagated1])
				}
				if from[propagated1] != "" {
					concurrentValues = append(concurrentValues, from[propagated1])
				}
				if add[propagated1] != "" {
					concurrentValues = append(concurrentValues, add[propagated1])
				}

				if len(concurrentValues) > 1 {
					testcases = append(testcases, TestPropagateFailsTestCase{
						LabelsToPropagate: []string{propagated1},

						To:   to,
						From: from,
						Add:  add,

						ExpectedOrigValue:      concurrentValues[0],
						ExpectedOverwriteValue: concurrentValues[1],
					})
				}
			}
		}
	}

	// execute
	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			tc := deepcopy.Copy(tc).(TestPropagateFailsTestCase)
			t.Parallel()
			logTestCaseSpec(t, tc)

			// SETUP
			toObj := &DummyObject1{}
			toObj.SetLabels(tc.To)

			fromObj := &DummyObject1{}
			fromObj.SetLabels(tc.From)

			// EXERCISE
			resultErr := propagate(tc.LabelsToPropagate, toObj, fromObj, tc.Add)

			// VERIFY
			assert.Error(t, resultErr, fmt.Sprintf(
				"label %q: cannot overwrite existing value %q with %q",
				propagated1, tc.ExpectedOrigValue, tc.ExpectedOverwriteValue,
			))
		})
	}
}

type TestLabelAsSystemManagedTestCase struct {
	// fields must be exported to make deepcopy work
	Orig     map[string]string
	Expected map[string]string
}

func Test__LabelAsSystemManaged(t *testing.T) {
	testcases := []TestLabelAsSystemManagedTestCase{}

	// generate testcases
	for _, orig := range []map[string]string{
		nil,
		{},
		{stewardv1alpha1.LabelSystemManaged: ""},
		{stewardv1alpha1.LabelSystemManaged: "someValue"},
	} {
		testcases = append(testcases, TestLabelAsSystemManagedTestCase{
			Orig: orig,
			Expected: map[string]string{
				stewardv1alpha1.LabelSystemManaged: "",
			},
		})
	}

	for _, orig := range []map[string]string{
		{
			"key1": "value1",
		},
		{
			"key1":                             "value1",
			stewardv1alpha1.LabelSystemManaged: "",
		},
		{
			"key1":                             "value1",
			stewardv1alpha1.LabelSystemManaged: "someValue",
		},
	} {
		testcases = append(testcases, TestLabelAsSystemManagedTestCase{
			Orig: orig,
			Expected: map[string]string{
				"key1":                             "value1",
				stewardv1alpha1.LabelSystemManaged: "",
			},
		})
	}

	// execute
	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			tc := deepcopy.Copy(tc).(TestLabelAsSystemManagedTestCase)
			t.Parallel()
			logTestCaseSpec(t, tc)

			// SETUP
			obj := &DummyObject1{}
			obj.SetLabels(tc.Orig)

			// EXERCISE
			LabelAsSystemManaged(obj)

			// VERIFY
			assert.DeepEqual(t, tc.Expected, obj.GetLabels())
		})
	}
}

func Test__LabelAsSystemManaged__NilArg(t *testing.T) {
	// EXERCISE
	LabelAsSystemManaged(nil)
}

func Test__LabelAsOwnedByClientNamespace__Success(t *testing.T) {
	const (
		clientNamespace1Name = "client-namespace1"
	)

	testcases := []struct {
		OrigLabels map[string]string
	}{
		{
			OrigLabels: nil,
		},
		{
			OrigLabels: map[string]string{},
		},
		{
			OrigLabels: map[string]string{
				stewardv1alpha1.LabelOwnerClientName:      clientNamespace1Name,
				stewardv1alpha1.LabelOwnerClientNamespace: clientNamespace1Name,
			},
		},
	}

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			tc := tc
			t.Parallel()

			// SETUP
			obj := &DummyObject1{}
			obj.SetLabels(tc.OrigLabels)

			owner := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: clientNamespace1Name,
				},
			}

			// EXERCISE
			resultErr := LabelAsOwnedByClientNamespace(obj, owner)

			// VERIFY
			assert.NilError(t, resultErr)

			expected := map[string]string{
				stewardv1alpha1.LabelOwnerClientName:      clientNamespace1Name,
				stewardv1alpha1.LabelOwnerClientNamespace: clientNamespace1Name,
			}
			assert.DeepEqual(t, expected, obj.GetLabels())
		})
	}
}

type TestLabelAsOwnedByXXXFailsOnOverwriteTestCase struct {
	OrigLabels    map[string]string
	ConflictLabel string
	ConflictValue string
}

func Test__LabelAsOwnedByClientNamespace__FailsOnOverwrite(t *testing.T) {
	const (
		clientNamespace1Name    = "client-namespace1"
		differentExistingValue1 = "otherExistingValue1"
	)

	testcases := []TestLabelAsOwnedByXXXFailsOnOverwriteTestCase{}

	// generate testcases
	allLabelsWithoutConflicts := map[string]string{
		stewardv1alpha1.LabelOwnerClientName:      clientNamespace1Name,
		stewardv1alpha1.LabelOwnerClientNamespace: clientNamespace1Name,
	}

	for key := range allLabelsWithoutConflicts {
		labels := deepcopy.Copy(allLabelsWithoutConflicts).(map[string]string)
		labels[key] = differentExistingValue1

		testcases = append(testcases, TestLabelAsOwnedByXXXFailsOnOverwriteTestCase{
			OrigLabels:    labels,
			ConflictLabel: key,
			ConflictValue: allLabelsWithoutConflicts[key],
		})
	}

	// run testcases
	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			tc := tc
			t.Parallel()

			// SETUP
			obj := &DummyObject1{}
			obj.SetLabels(deepcopy.Copy(tc.OrigLabels).(map[string]string))

			owner := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: clientNamespace1Name,
				},
			}

			// EXERCISE
			resultErr := LabelAsOwnedByClientNamespace(obj, owner)

			// VERIFY
			assert.Error(t, resultErr,
				fmt.Sprintf(
					"label %q: cannot overwrite existing value %q with %q",
					tc.ConflictLabel, differentExistingValue1, tc.ConflictValue))

			assert.DeepEqual(t, tc.OrigLabels, obj.GetLabels())
		})
	}
}

func Test__LabelAsOwnedByClientNamespace__NilArg__obj(t *testing.T) {
	// SETUP
	owner := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace1",
		},
	}

	// EXERCISE
	resultErr := LabelAsOwnedByClientNamespace(nil, owner)

	// VERIFY
	assert.NilError(t, resultErr)
}

func Test__LabelAsOwnedByClientNamespace__NilArg__owner(t *testing.T) {
	// SETUP
	obj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace1",
			Labels: map[string]string{
				stewardv1alpha1.LabelOwnerClientName:      "client1",
				stewardv1alpha1.LabelOwnerClientNamespace: "client-namespace1",
				"other": "other1",
			},
		},
	}
	objCopy := obj.DeepCopy()

	// EXERCISE
	assert.Assert(t, cmp.Panics(func() {
		LabelAsOwnedByClientNamespace(obj, nil)
	}))

	// VERIFY
	assert.DeepEqual(t, objCopy, obj) // no modification
}

func Test__LabelAsOwnedByTenant__Success(t *testing.T) {
	const (
		clientNamespace1Name = "client-namespace1"
		tenant1Name          = "tenant1"
		tenant1Namespace     = "tenant-namespace1"
	)

	testcases := []struct {
		OrigLabels map[string]string
	}{
		{
			OrigLabels: nil,
		},
		{
			OrigLabels: map[string]string{},
		},
		{
			OrigLabels: map[string]string{
				stewardv1alpha1.LabelOwnerClientName:      clientNamespace1Name,
				stewardv1alpha1.LabelOwnerClientNamespace: clientNamespace1Name,
				stewardv1alpha1.LabelOwnerTenantName:      tenant1Name,
				stewardv1alpha1.LabelOwnerTenantNamespace: tenant1Namespace,
			},
		},
	}

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			tc := tc
			t.Parallel()

			// SETUP
			obj := &DummyObject1{}
			obj.SetLabels(tc.OrigLabels)

			owner := &stewardv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tenant1Name,
					Namespace: clientNamespace1Name,
				},
				Status: stewardv1alpha1.TenantStatus{
					TenantNamespaceName: tenant1Namespace,
				},
			}

			// EXERCISE
			resultErr := LabelAsOwnedByTenant(obj, owner)

			// VERIFY
			assert.NilError(t, resultErr)

			expected := map[string]string{
				stewardv1alpha1.LabelOwnerClientName:      clientNamespace1Name,
				stewardv1alpha1.LabelOwnerClientNamespace: clientNamespace1Name,
				stewardv1alpha1.LabelOwnerTenantName:      tenant1Name,
				stewardv1alpha1.LabelOwnerTenantNamespace: tenant1Namespace,
			}
			assert.DeepEqual(t, expected, obj.GetLabels())
		})
	}
}

func Test__LabelAsOwnedByTenant__FailsOnOverwrite(t *testing.T) {
	const (
		clientNamespace1Name    = "client-namespace1"
		tenant1Name             = "tenant1"
		tenant1Namespace        = "tenant-namespace1"
		differentExistingValue1 = "otherExistingValue1"
	)

	testcases := []TestLabelAsOwnedByXXXFailsOnOverwriteTestCase{}

	// generate testcases
	allLabelsWithoutConflicts := map[string]string{
		stewardv1alpha1.LabelOwnerClientName:      clientNamespace1Name,
		stewardv1alpha1.LabelOwnerClientNamespace: clientNamespace1Name,
		stewardv1alpha1.LabelOwnerTenantName:      tenant1Name,
		stewardv1alpha1.LabelOwnerTenantNamespace: tenant1Namespace,
	}

	for key := range allLabelsWithoutConflicts {
		labels := deepcopy.Copy(allLabelsWithoutConflicts).(map[string]string)
		labels[key] = differentExistingValue1

		testcases = append(testcases, TestLabelAsOwnedByXXXFailsOnOverwriteTestCase{
			OrigLabels:    labels,
			ConflictLabel: key,
			ConflictValue: allLabelsWithoutConflicts[key],
		})
	}

	// run testcases
	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			tc := tc
			t.Parallel()

			// SETUP
			obj := &DummyObject1{}
			obj.SetLabels(deepcopy.Copy(tc.OrigLabels).(map[string]string))

			owner := &stewardv1alpha1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tenant1Name,
					Namespace: clientNamespace1Name,
				},
				Status: stewardv1alpha1.TenantStatus{
					TenantNamespaceName: tenant1Namespace,
				},
			}

			// EXERCISE
			resultErr := LabelAsOwnedByTenant(obj, owner)

			// VERIFY
			assert.Error(t, resultErr,
				fmt.Sprintf(
					"label %q: cannot overwrite existing value %q with %q",
					tc.ConflictLabel, differentExistingValue1, tc.ConflictValue))

			assert.DeepEqual(t, tc.OrigLabels, obj.GetLabels())
		})
	}
}

func Test__LabelAsOwnedByTenant__NilArg__obj(t *testing.T) {
	// SETUP
	owner := &stewardv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tenant1",
			Namespace: "namespace1",
		},
	}

	// EXERCISE
	resultErr := LabelAsOwnedByTenant(nil, owner)

	// VERIFY
	assert.NilError(t, resultErr)
}

func Test__LabelAsOwnedByTenant__NilArg__owner(t *testing.T) {
	// SETUP
	obj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace1",
			Labels: map[string]string{
				stewardv1alpha1.LabelOwnerClientName:      "client1",
				stewardv1alpha1.LabelOwnerClientNamespace: "client-namespace1",
				stewardv1alpha1.LabelOwnerTenantName:      "tenant1",
				stewardv1alpha1.LabelOwnerTenantNamespace: "tenant-namespace1",
				"other": "other1",
			},
		},
	}
	objCopy := obj.DeepCopy()

	// EXERCISE
	assert.Assert(t, cmp.Panics(func() {
		LabelAsOwnedByTenant(obj, nil)
	}))

	// VERIFY
	assert.DeepEqual(t, objCopy, obj) // no modification
}

func Test__LabelAsOwnedByPipelineRun__Success(t *testing.T) {
	const (
		clientNamespace1Name = "client-namespace1"
		tenant1Name          = "tenant1"
		tenant1Namespace     = "tenant-namespace1"
		pipelinerun1Name     = "pipelinerun1"
	)

	testcases := []struct {
		OrigLabels map[string]string
	}{
		{
			OrigLabels: nil,
		},
		{
			OrigLabels: map[string]string{},
		},
		{
			OrigLabels: map[string]string{
				stewardv1alpha1.LabelOwnerClientName:      clientNamespace1Name,
				stewardv1alpha1.LabelOwnerClientNamespace: clientNamespace1Name,
				stewardv1alpha1.LabelOwnerTenantName:      tenant1Name,
				stewardv1alpha1.LabelOwnerTenantNamespace: tenant1Namespace,
				stewardv1alpha1.LabelOwnerPipelineRunName: pipelinerun1Name,
			},
		},
	}

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			tc := tc
			t.Parallel()

			// SETUP
			obj := &DummyObject1{}
			obj.SetLabels(tc.OrigLabels)

			owner := &stewardv1alpha1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pipelinerun1Name,
					Namespace: tenant1Namespace,
					Labels: map[string]string{
						stewardv1alpha1.LabelOwnerClientName:      clientNamespace1Name,
						stewardv1alpha1.LabelOwnerClientNamespace: clientNamespace1Name,
						stewardv1alpha1.LabelOwnerTenantName:      tenant1Name,
						// stewardv1alpha1.LabelOwnerTenantNamespace is derived from Namespace
					},
				},
			}

			// EXERCISE
			resultErr := LabelAsOwnedByPipelineRun(obj, owner)

			// VERIFY
			assert.NilError(t, resultErr)

			expected := map[string]string{
				stewardv1alpha1.LabelOwnerClientName:      clientNamespace1Name,
				stewardv1alpha1.LabelOwnerClientNamespace: clientNamespace1Name,
				stewardv1alpha1.LabelOwnerTenantName:      tenant1Name,
				stewardv1alpha1.LabelOwnerTenantNamespace: tenant1Namespace,
				stewardv1alpha1.LabelOwnerPipelineRunName: pipelinerun1Name,
			}
			assert.DeepEqual(t, expected, obj.GetLabels())
		})
	}
}

func Test__LabelAsOwnedByPipelineRun__FailsOnOverwrite(t *testing.T) {
	const (
		clientNamespace1Name    = "client-namespace1"
		tenant1Name             = "tenant1"
		tenant1Namespace        = "tenant-namespace1"
		pipelinerun1Name        = "pipelinerun1"
		differentExistingValue1 = "otherExistingValue1"
	)

	testcases := []TestLabelAsOwnedByXXXFailsOnOverwriteTestCase{}

	// generate testcases
	allLabelsWithoutConflicts := map[string]string{
		stewardv1alpha1.LabelOwnerClientName:      clientNamespace1Name,
		stewardv1alpha1.LabelOwnerClientNamespace: clientNamespace1Name,
		stewardv1alpha1.LabelOwnerTenantName:      tenant1Name,
		stewardv1alpha1.LabelOwnerTenantNamespace: tenant1Namespace,
	}

	for key := range allLabelsWithoutConflicts {
		labels := deepcopy.Copy(allLabelsWithoutConflicts).(map[string]string)
		labels[key] = differentExistingValue1

		testcases = append(testcases, TestLabelAsOwnedByXXXFailsOnOverwriteTestCase{
			OrigLabels:    labels,
			ConflictLabel: key,
			ConflictValue: allLabelsWithoutConflicts[key],
		})
	}

	// run testcases
	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			tc := tc
			t.Parallel()

			// SETUP
			obj := &DummyObject1{}
			obj.SetLabels(deepcopy.Copy(tc.OrigLabels).(map[string]string))

			owner := &stewardv1alpha1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pipelinerun1Name,
					Namespace: tenant1Namespace,
					Labels: map[string]string{
						stewardv1alpha1.LabelOwnerClientName:      clientNamespace1Name,
						stewardv1alpha1.LabelOwnerClientNamespace: clientNamespace1Name,
						stewardv1alpha1.LabelOwnerTenantName:      tenant1Name,
						// stewardv1alpha1.LabelOwnerTenantNamespace is derived from Namespace
					},
				},
			}

			// EXERCISE
			resultErr := LabelAsOwnedByPipelineRun(obj, owner)

			// VERIFY
			assert.Error(t, resultErr,
				fmt.Sprintf(
					"label %q: cannot overwrite existing value %q with %q",
					tc.ConflictLabel, differentExistingValue1, tc.ConflictValue))

			assert.DeepEqual(t, tc.OrigLabels, obj.GetLabels())
		})
	}
}

func Test__LabelAsOwnedByPipelineRun__NilArg__obj(t *testing.T) {
	// SETUP
	owner := &stewardv1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pipelinerun1",
			Namespace: "namespace1",
		},
	}

	// EXERCISE
	resultErr := LabelAsOwnedByPipelineRun(nil, owner)

	// VERIFY
	assert.NilError(t, resultErr)
}

func Test__LabelAsOwnedByPipelineRun__NilArg__owner(t *testing.T) {
	// SETUP
	obj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "namespace1",
			Labels: map[string]string{
				stewardv1alpha1.LabelOwnerClientName:      "client1",
				stewardv1alpha1.LabelOwnerClientNamespace: "client-namespace1",
				stewardv1alpha1.LabelOwnerTenantName:      "tenant1",
				stewardv1alpha1.LabelOwnerTenantNamespace: "tenant-namespace1",
				stewardv1alpha1.LabelOwnerPipelineRunName: "pipelinerun1",
				"other": "other1",
			},
		},
	}
	objCopy := obj.DeepCopy()

	// EXERCISE
	assert.Assert(t, cmp.Panics(func() {
		LabelAsOwnedByPipelineRun(obj, nil)
	}))

	// VERIFY
	assert.DeepEqual(t, objCopy, obj) // no modification
}

type TestHelper1 struct {
	t *testing.T
}

func (h *TestHelper1) propagatedLabelsList() []string {
	t := h.t
	t.Helper()

	return []string{
		"propagated1",
		"propagated2",
	}
}

type DummyObject1 struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func logTestCaseSpec(t *testing.T, spec interface{}) {
	t.Helper()

	spewConf := &spew.ConfigState{
		Indent:                  "\t",
		DisableCapacities:       true,
		DisablePointerAddresses: true,
	}
	t.Logf("Testcase:\n%s", spewConf.Sdump(spec))
}
