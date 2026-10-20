package env

import "testing"

func TestMemoryStoreRoundTrip(t *testing.T) {
	var s MemoryStore // zero value is usable
	beta := NewEnvironment("beta", "")
	beta.SetVariable("BASE_URL", "https://beta.example.com")
	if err := s.SaveEnv(beta); err != nil {
		t.Fatalf("SaveEnv: %v", err)
	}
	beta.SetVariable("BASE_URL", "mutated")

	got, err := s.GetEnvByName("beta")
	if err != nil {
		t.Fatalf("GetEnvByName: %v", err)
	}
	if got.Variables["BASE_URL"] != "https://beta.example.com" {
		t.Fatalf("stored env aliased caller's value: %+v", got.Variables)
	}
	if byID, err := s.GetEnv(got.ID); err != nil || byID.Name != "beta" {
		t.Fatalf("GetEnv = %+v, %v", byID, err)
	}

	if err := s.SetActiveEnv("beta"); err != nil {
		t.Fatalf("SetActiveEnv: %v", err)
	}
	if active, _ := s.GetActiveEnv(); active != "beta" {
		t.Fatalf("GetActiveEnv = %q, want beta", active)
	}

	if err := s.DeleteEnv(got.ID); err != nil {
		t.Fatalf("DeleteEnv: %v", err)
	}
	if _, err := s.GetEnvByName("beta"); err == nil {
		t.Fatal("GetEnvByName after delete succeeded")
	}
	if envs, _ := s.ListEnvs(); len(envs) != 0 {
		t.Fatalf("ListEnvs after delete = %+v", envs)
	}
}
