package scripting

type ChainExecutor struct {
	engine         *Engine
	visited        map[string]int
	iterationCount int
	maxIterations  int
}

type ChainExecutorOption func(*ChainExecutor)

func NewChainExecutor(engine *Engine, opts ...ChainExecutorOption) *ChainExecutor {
	ce := &ChainExecutor{
		engine:        engine,
		visited:       make(map[string]int),
		maxIterations: 100,
	}
	for _, opt := range opts {
		opt(ce)
	}
	return ce
}

func WithMaxIterations(max int) ChainExecutorOption {
	return func(ce *ChainExecutor) {
		ce.maxIterations = max
	}
}

func (ce *ChainExecutor) MarkIteration(requestName string) {
	if requestName == "" {
		return
	}
	ce.iterationCount++
	ce.visited[requestName]++
}

func (ce *ChainExecutor) GetNextRequest() string {
	if ce.engine == nil {
		return ""
	}
	return ce.engine.nextRequest
}

func (ce *ChainExecutor) IsCircular() bool {
	for _, count := range ce.visited {
		if count >= 3 {
			return true
		}
	}
	return false
}

func (ce *ChainExecutor) MaxIterations() int {
	return ce.maxIterations
}

func (ce *ChainExecutor) MaxIterationsReached() bool {
	return ce.iterationCount >= ce.maxIterations
}

func (ce *ChainExecutor) Reset() {
	ce.visited = make(map[string]int)
	ce.iterationCount = 0
}

func (ce *ChainExecutor) Variables() map[string]string {
	if ce.engine == nil {
		return nil
	}
	return ce.engine.variables
}
