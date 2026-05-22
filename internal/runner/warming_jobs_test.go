package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"aquecedor-evolution/backend/internal/evolution"
	"aquecedor-evolution/backend/internal/executor"
	"aquecedor-evolution/backend/internal/repository"
)

type fakeRunnerJobStore struct {
	job     repository.WarmingJob
	updated []runnerStatusUpdate
}

func (s fakeRunnerJobStore) GetByID(ctx context.Context, id string) (repository.WarmingJob, error) {
	return s.job, nil
}

type runnerStatusUpdate struct {
	id     string
	status string
	err    string
}

func (s *fakeRunnerJobStore) UpdateStatus(ctx context.Context, id string, status string, errorText string) error {
	s.updated = append(s.updated, runnerStatusUpdate{id: id, status: status, err: errorText})
	return nil
}

type fakeRunnerStepStore struct {
	steps []repository.ConversationStep
}

func (s fakeRunnerStepStore) ListByScriptID(ctx context.Context, scriptID string) ([]repository.ConversationStep, error) {
	return s.steps, nil
}

type fakeRunnerInstanceStore struct {
	instances  map[string]repository.Instance
	requests   []string
	errByPhone map[string]error
}

func (s *fakeRunnerInstanceStore) GetOpenByPhoneNumberID(ctx context.Context, phoneNumberID string) (repository.Instance, error) {
	s.requests = append(s.requests, phoneNumberID)
	if err := s.errByPhone[phoneNumberID]; err != nil {
		return repository.Instance{}, err
	}
	return s.instances[phoneNumberID], nil
}

type fakeRunnerStepExecutor struct {
	calls       []runnerExecutionCall
	err         error
	textRequest evolution.SendTextRequest
}

type runnerExecutionCall struct {
	instanceName string
	stepOrder    int
	payload      map[string]any
}

func (e *fakeRunnerStepExecutor) Execute(ctx context.Context, instanceName string, step repository.ConversationStep) (executor.StepResult, error) {
	e.calls = append(e.calls, runnerExecutionCall{instanceName: instanceName, stepOrder: step.StepOrder, payload: step.Payload})
	if e.err != nil {
		return executor.StepResult{}, e.err
	}
	return executor.StepResult{MessageKey: &evolution.MessageKey{ID: "message-id"}}, nil
}

func TestWarmingJobRunnerResolvesPayloadPlaceholders(t *testing.T) {
	scriptID := "script-id"
	job := repository.WarmingJob{
		ID:       "job-id",
		ScriptID: &scriptID,
		PhoneAID: "phone-a-id",
		PhoneBID: "phone-b-id",
		Metadata: map[string]any{
			"phoneAE164":       "5511888888888",
			"phoneBE164":       "5511999999999",
			"triggerMessageId": "trigger-message-id",
			"triggerRemoteJid": "5511999999999@s.whatsapp.net",
		},
	}
	instanceStore := &fakeRunnerInstanceStore{instances: map[string]repository.Instance{
		"phone-a-id": {ID: "instance-a-id", InstanceName: "chip-a"},
		"phone-b-id": {ID: "instance-b-id", InstanceName: "chip-b"},
	}}
	stepExecutor := &fakeRunnerStepExecutor{}
	runner := NewWarmingJobRunner(
		&fakeRunnerJobStore{job: job},
		fakeRunnerStepStore{steps: []repository.ConversationStep{
			{
				ID:         "step-1",
				ScriptID:   scriptID,
				StepOrder:  1,
				SenderRole: "a",
				ActionType: "send_reply",
				Payload: map[string]any{
					"number":    "{{phoneB}}",
					"text":      "ok",
					"remoteJid": "{{triggerRemoteJid}}",
					"messageId": "{{triggerMessageId}}",
				},
			},
		}},
		instanceStore,
		fakeRunnerExecutorFactory{byInstance: map[string]*fakeRunnerStepExecutor{
			"instance-a-id": stepExecutor,
			"instance-b-id": stepExecutor,
		}},
		&fakeRunnerExecutionLogStore{},
		&fakeConcurrencyGate{},
		nil,
		nil,
	)

	if _, err := runner.Run(context.Background(), "job-id"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if stepExecutor.textRequest.Number != "5511999999999" {
		t.Fatalf("number = %q", stepExecutor.textRequest.Number)
	}
	if stepExecutor.textRequest.Quoted == nil || stepExecutor.textRequest.Quoted.Key.RemoteJID != "5511999999999@s.whatsapp.net" {
		t.Fatalf("remoteJid = %#v", stepExecutor.textRequest.Quoted)
	}
	if stepExecutor.textRequest.Quoted == nil || stepExecutor.textRequest.Quoted.Key.ID != "trigger-message-id" {
		t.Fatalf("messageId = %#v", stepExecutor.textRequest.Quoted)
	}
}

func (e *fakeRunnerStepExecutor) SendText(ctx context.Context, instanceName string, request evolution.SendTextRequest) (evolution.SendMessageResponse, error) {
	e.calls = append(e.calls, runnerExecutionCall{instanceName: instanceName})
	e.textRequest = request
	if e.err != nil {
		return evolution.SendMessageResponse{}, e.err
	}
	return evolution.SendMessageResponse{Key: evolution.MessageKey{ID: "message-id"}}, nil
}

func (e *fakeRunnerStepExecutor) SendMedia(ctx context.Context, instanceName string, request evolution.SendMediaRequest) (evolution.SendMessageResponse, error) {
	e.calls = append(e.calls, runnerExecutionCall{instanceName: instanceName})
	if e.err != nil {
		return evolution.SendMessageResponse{}, e.err
	}
	return evolution.SendMessageResponse{Key: evolution.MessageKey{ID: "media-id"}}, nil
}

func (e *fakeRunnerStepExecutor) SendWhatsAppAudio(ctx context.Context, instanceName string, request evolution.SendWhatsAppAudioRequest) (evolution.SendMessageResponse, error) {
	e.calls = append(e.calls, runnerExecutionCall{instanceName: instanceName})
	if e.err != nil {
		return evolution.SendMessageResponse{}, e.err
	}
	return evolution.SendMessageResponse{Key: evolution.MessageKey{ID: "audio-id"}}, nil
}

func (e *fakeRunnerStepExecutor) SendStatus(ctx context.Context, instanceName string, request evolution.SendStatusRequest) (evolution.SendMessageResponse, error) {
	e.calls = append(e.calls, runnerExecutionCall{instanceName: instanceName})
	if e.err != nil {
		return evolution.SendMessageResponse{}, e.err
	}
	return evolution.SendMessageResponse{Key: evolution.MessageKey{ID: "status-id"}}, nil
}

func (e *fakeRunnerStepExecutor) SendSticker(ctx context.Context, instanceName string, request evolution.SendStickerRequest) error {
	e.calls = append(e.calls, runnerExecutionCall{instanceName: instanceName})
	return e.err
}

func (e *fakeRunnerStepExecutor) SendPresence(ctx context.Context, instanceName string, request evolution.SendPresenceRequest) error {
	e.calls = append(e.calls, runnerExecutionCall{instanceName: instanceName})
	return e.err
}

func (e *fakeRunnerStepExecutor) SendReaction(ctx context.Context, instanceName string, request evolution.SendReactionRequest) error {
	e.calls = append(e.calls, runnerExecutionCall{instanceName: instanceName})
	return e.err
}

type fakeRunnerExecutorFactory struct {
	byInstance map[string]*fakeRunnerStepExecutor
}

func (f fakeRunnerExecutorFactory) ForInstance(ctx context.Context, instance repository.Instance) (*executor.StepExecutor, error) {
	target := f.byInstance[instance.ID]
	stepExecutor := executor.NewStepExecutor(target)
	return &stepExecutor, nil
}

type fakeConcurrencyGate struct {
	err      error
	serverID string
	phoneAID string
	phoneBID string
}

func (g *fakeConcurrencyGate) Check(ctx context.Context, serverID string, phoneAID string, phoneBID string) error {
	g.serverID = serverID
	g.phoneAID = phoneAID
	g.phoneBID = phoneBID
	return g.err
}

type fakeRunnerExecutionLogStore struct {
	params          []repository.CreateExecutionLogParams
	successfulSteps map[string]bool
}

func (s *fakeRunnerExecutionLogStore) Create(ctx context.Context, params repository.CreateExecutionLogParams) (repository.ExecutionLog, error) {
	s.params = append(s.params, params)
	return repository.ExecutionLog{ID: "log-id", Status: params.Status}, nil
}

func (s *fakeRunnerExecutionLogStore) ExistsSuccessfulStep(ctx context.Context, warmingJobID string, stepID string) (bool, error) {
	return s.successfulSteps[stepID], nil
}

func TestWarmingJobRunnerExecutesStepsAndLogsSuccess(t *testing.T) {
	scriptID := "script-id"
	job := repository.WarmingJob{
		ID:          "job-id",
		ScriptID:    &scriptID,
		PhoneAID:    "phone-a-id",
		PhoneBID:    "phone-b-id",
		ScheduledAt: time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC),
	}
	instanceStore := &fakeRunnerInstanceStore{instances: map[string]repository.Instance{
		"phone-a-id": {ID: "instance-a-id", InstanceName: "chip-a"},
		"phone-b-id": {ID: "instance-b-id", InstanceName: "chip-b"},
	}}
	stepExecutorA := &fakeRunnerStepExecutor{}
	stepExecutorB := &fakeRunnerStepExecutor{}
	logStore := &fakeRunnerExecutionLogStore{}
	jobStore := &fakeRunnerJobStore{job: job}
	gate := &fakeConcurrencyGate{}
	runner := NewWarmingJobRunner(
		jobStore,
		fakeRunnerStepStore{steps: []repository.ConversationStep{
			{ID: "step-1", ScriptID: scriptID, StepOrder: 1, SenderRole: "a", ActionType: "send_presence", Payload: map[string]any{}},
			{ID: "step-2", ScriptID: scriptID, StepOrder: 2, SenderRole: "b", ActionType: "send_text", Payload: map[string]any{}},
		}},
		instanceStore,
		fakeRunnerExecutorFactory{byInstance: map[string]*fakeRunnerStepExecutor{
			"instance-a-id": stepExecutorA,
			"instance-b-id": stepExecutorB,
		}},
		logStore,
		gate,
		nil,
		nil,
	)

	executed, err := runner.Run(context.Background(), "job-id")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if executed != 2 {
		t.Fatalf("executed = %d", executed)
	}
	if stepExecutorA.calls[0].instanceName != "chip-a" {
		t.Fatalf("call 0 instance = %q", stepExecutorA.calls[0].instanceName)
	}
	if stepExecutorB.calls[0].instanceName != "chip-b" {
		t.Fatalf("call 1 instance = %q", stepExecutorB.calls[0].instanceName)
	}
	if len(logStore.params) != 2 {
		t.Fatalf("logs len = %d", len(logStore.params))
	}
	if logStore.params[0].Status != "success" {
		t.Fatalf("log 0 status = %q", logStore.params[0].Status)
	}
	if len(logStore.params[0].EvolutionMessageKey) != 0 {
		t.Fatalf("log 0 message key = %v", logStore.params[0].EvolutionMessageKey)
	}
	if logStore.params[1].EvolutionMessageKey["id"] != "message-id" {
		t.Fatalf("log 1 message key = %v", logStore.params[1].EvolutionMessageKey)
	}
	if logStore.params[0].InstanceID == nil || *logStore.params[0].InstanceID != "instance-a-id" {
		t.Fatalf("log 0 instance = %v", logStore.params[0].InstanceID)
	}
	if len(jobStore.updated) != 2 || jobStore.updated[0].status != "running" || jobStore.updated[1].status != "success" {
		t.Fatalf("job updates = %+v", jobStore.updated)
	}
	if gate.serverID != "" {
		// checked below in a dedicated test where server ids are set
	}
}

func TestWarmingJobRunnerLogsFailure(t *testing.T) {
	scriptID := "script-id"
	logStore := &fakeRunnerExecutionLogStore{}
	jobStore := &fakeRunnerJobStore{job: repository.WarmingJob{ID: "job-id", ScriptID: &scriptID, PhoneAID: "phone-a-id", PhoneBID: "phone-b-id"}}
	runner := NewWarmingJobRunner(
		jobStore,
		fakeRunnerStepStore{steps: []repository.ConversationStep{{ID: "step-1", StepOrder: 1, SenderRole: "a", ActionType: "send_text"}}},
		&fakeRunnerInstanceStore{instances: map[string]repository.Instance{"phone-a-id": {ID: "instance-a-id", InstanceName: "chip-a"}}},
		fakeRunnerExecutorFactory{byInstance: map[string]*fakeRunnerStepExecutor{
			"instance-a-id": {err: errors.New("evolution failed")},
		}},
		logStore,
		&fakeConcurrencyGate{},
		nil,
		nil,
	)

	_, err := runner.Run(context.Background(), "job-id")
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if len(logStore.params) != 1 {
		t.Fatalf("logs len = %d", len(logStore.params))
	}
	if logStore.params[0].Status != "failed" {
		t.Fatalf("status = %q", logStore.params[0].Status)
	}
	if logStore.params[0].Error != "evolution failed" {
		t.Fatalf("error = %q", logStore.params[0].Error)
	}
	if len(jobStore.updated) != 2 || jobStore.updated[1].status != "failed" {
		t.Fatalf("job updates = %+v", jobStore.updated)
	}
}

func TestWarmingJobRunnerMarksFailedWhenInstanceLookupFails(t *testing.T) {
	scriptID := "script-id"
	jobStore := &fakeRunnerJobStore{job: repository.WarmingJob{ID: "job-id", ScriptID: &scriptID, PhoneAID: "phone-a-id", PhoneBID: "phone-b-id"}}
	runner := NewWarmingJobRunner(
		jobStore,
		fakeRunnerStepStore{steps: []repository.ConversationStep{{ID: "step-1", StepOrder: 1, SenderRole: "a", ActionType: "send_text"}}},
		&fakeRunnerInstanceStore{errByPhone: map[string]error{"phone-a-id": errors.New("no open instance")}},
		fakeRunnerExecutorFactory{},
		&fakeRunnerExecutionLogStore{},
		&fakeConcurrencyGate{},
		nil,
		nil,
	)

	_, err := runner.Run(context.Background(), "job-id")
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if len(jobStore.updated) != 1 || jobStore.updated[0].status != "failed" {
		t.Fatalf("job updates = %+v", jobStore.updated)
	}
	if jobStore.updated[0].err != "no open instance" {
		t.Fatalf("error = %q", jobStore.updated[0].err)
	}
}

func TestWarmingJobRunnerSkipsAlreadySuccessfulStep(t *testing.T) {
	scriptID := "script-id"
	stepExecutor := &fakeRunnerStepExecutor{}
	logStore := &fakeRunnerExecutionLogStore{successfulSteps: map[string]bool{"step-1": true}}
	runner := NewWarmingJobRunner(
		&fakeRunnerJobStore{job: repository.WarmingJob{ID: "job-id", ScriptID: &scriptID, PhoneAID: "phone-a-id", PhoneBID: "phone-b-id"}},
		fakeRunnerStepStore{steps: []repository.ConversationStep{{ID: "step-1", StepOrder: 1, SenderRole: "a", ActionType: "send_text"}}},
		&fakeRunnerInstanceStore{instances: map[string]repository.Instance{"phone-a-id": {ID: "instance-a-id", InstanceName: "chip-a"}, "phone-b-id": {ID: "instance-b-id", InstanceName: "chip-b"}}},
		fakeRunnerExecutorFactory{byInstance: map[string]*fakeRunnerStepExecutor{"instance-a-id": stepExecutor, "instance-b-id": {}}},
		logStore,
		&fakeConcurrencyGate{},
		nil,
		nil,
	)

	executed, err := runner.Run(context.Background(), "job-id")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if executed != 0 {
		t.Fatalf("executed = %d", executed)
	}
	if len(stepExecutor.calls) != 0 {
		t.Fatalf("calls len = %d", len(stepExecutor.calls))
	}
	if len(logStore.params) != 0 {
		t.Fatalf("logs len = %d", len(logStore.params))
	}
}

func TestWarmingJobRunnerLogsAsyncAcceptedStatus(t *testing.T) {
	scriptID := "script-id"
	job := repository.WarmingJob{
		ID:       "job-id",
		ScriptID: &scriptID,
		PhoneAID: "phone-a-id",
		PhoneBID: "phone-b-id",
	}
	instanceStore := &fakeRunnerInstanceStore{instances: map[string]repository.Instance{
		"phone-a-id": {ID: "instance-a-id", InstanceName: "chip-a"},
		"phone-b-id": {ID: "instance-b-id", InstanceName: "chip-b"},
	}}
	logStore := &fakeRunnerExecutionLogStore{}
	jobStore := &fakeRunnerJobStore{job: job}
	runner := NewWarmingJobRunner(
		jobStore,
		fakeRunnerStepStore{steps: []repository.ConversationStep{
			{ID: "step-1", ScriptID: scriptID, StepOrder: 1, SenderRole: "a", ActionType: "send_status", Payload: map[string]any{"type": "text", "content": "Oi", "allContacts": true}},
		}},
		instanceStore,
		fakeRunnerExecutorFactory{byInstance: map[string]*fakeRunnerStepExecutor{
			"instance-a-id": {},
			"instance-b-id": {},
		}},
		logStore,
		&fakeConcurrencyGate{},
		nil,
		nil,
	)

	// swap explicit executor with async behavior for instance A
	asyncExec := &fakeRunnerStepExecutorAsyncStatus{}
	runner.executors = fakeRunnerExecutorFactoryAsync{byInstance: map[string]*fakeRunnerStepExecutorAsyncStatus{
		"instance-a-id": asyncExec,
		"instance-b-id": asyncExec,
	}}

	executed, err := runner.Run(context.Background(), "job-id")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if executed != 1 {
		t.Fatalf("executed = %d", executed)
	}
	if len(logStore.params) != 1 {
		t.Fatalf("logs len = %d", len(logStore.params))
	}
	if logStore.params[0].ResponsePayload["acceptedAsync"] != true {
		t.Fatalf("response payload = %+v", logStore.params[0].ResponsePayload)
	}
	if len(logStore.params[0].EvolutionMessageKey) != 0 {
		t.Fatalf("message key = %+v", logStore.params[0].EvolutionMessageKey)
	}
}

type fakeRunnerStepExecutorAsyncStatus struct {
	fakeRunnerStepExecutor
}

func (e *fakeRunnerStepExecutorAsyncStatus) SendStatus(ctx context.Context, instanceName string, request evolution.SendStatusRequest) (evolution.SendMessageResponse, error) {
	return evolution.SendMessageResponse{AcceptedAsync: true}, nil
}

type fakeRunnerExecutorFactoryAsync struct {
	byInstance map[string]*fakeRunnerStepExecutorAsyncStatus
}

func (f fakeRunnerExecutorFactoryAsync) ForInstance(ctx context.Context, instance repository.Instance) (*executor.StepExecutor, error) {
	target := f.byInstance[instance.ID]
	stepExecutor := executor.NewStepExecutor(target)
	return &stepExecutor, nil
}

func TestWarmingJobRunnerBlocksWhenConcurrencyGateFails(t *testing.T) {
	scriptID := "script-id"
	jobStore := &fakeRunnerJobStore{job: repository.WarmingJob{ID: "job-id", ScriptID: &scriptID, PhoneAID: "phone-a-id", PhoneBID: "phone-b-id"}}
	gate := &fakeConcurrencyGate{err: errors.New("instance concurrency exceeded")}
	runner := NewWarmingJobRunner(
		jobStore,
		fakeRunnerStepStore{steps: []repository.ConversationStep{{ID: "step-1", StepOrder: 1, SenderRole: "a", ActionType: "send_text"}}},
		&fakeRunnerInstanceStore{instances: map[string]repository.Instance{
			"phone-a-id": {ID: "instance-a-id", InstanceName: "chip-a", EvolutionServerID: "server-a"},
			"phone-b-id": {ID: "instance-b-id", InstanceName: "chip-b", EvolutionServerID: "server-a"},
		}},
		fakeRunnerExecutorFactory{byInstance: map[string]*fakeRunnerStepExecutor{
			"instance-a-id": {},
			"instance-b-id": {},
		}},
		&fakeRunnerExecutionLogStore{},
		gate,
		nil,
		nil,
	)

	_, err := runner.Run(context.Background(), "job-id")
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if gate.serverID != "server-a" {
		t.Fatalf("serverID = %q", gate.serverID)
	}
	if len(jobStore.updated) != 1 || jobStore.updated[0].status != "failed" {
		t.Fatalf("job updates = %+v", jobStore.updated)
	}
}
