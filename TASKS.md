# Task Tracker - Unified AWS Bedrock Controller

## Current Sprint: Bootstrap & Initial Agent Implementation

### ✅ Completed Tasks

### 🚧 In Progress Tasks

- [ ] **Bootstrap repo** (ID: bootstrap-repo)
  - **Goal**: Set up initial project structure with all necessary files
  - **Changed files**: Makefile, go.mod, .gitignore, README.md, TASKS.md
  - **Tests to add/update**: N/A (infrastructure setup)
  - **Exit criteria**: 
    - ✅ All files created and properly structured
    - ⏳ Git repository initialized with first commit
    - ⏳ `make dev-setup` runs successfully
    - ⏳ `make tidy` completes without errors

### 📋 Pending Tasks

- [ ] **Define initial CRD** (ID: define-initial-crd)
  - **Goal**: Create unified BedrockResource CRD that supports Agents, Inference Profiles, and Knowledge Bases
  - **Changed files**: /crd/bedrockresource.yaml
  - **Tests to add/update**: CRD validation tests
  - **Exit criteria**: 
    - CRD YAML with proper OpenAPI v3 schema validation
    - Status subresource enabled with standard Conditions
    - Support for polymorphic resource types (agent, inferenceProfile, knowledgeBase)
    - `make crd-validate` passes

- [ ] **Mirror types** (ID: mirror-types)
  - **Goal**: Create Go structs in /controller/types.go that mirror the CRD schema
  - **Changed files**: /controller/types.go
  - **Tests to add/update**: Type validation unit tests
  - **Exit criteria**:
    - Go structs with proper json tags match CRD schema exactly
    - Support for BedrockResourceSpec with polymorphic type field
    - BedrockResourceStatus with standard Kubernetes Conditions
    - `go build` succeeds for controller package

- [ ] **Create example CRs** (ID: example-cr)
  - **Goal**: Create sample Custom Resources for each supported type starting with Agent
  - **Changed files**: /cr/bedrockagent-sample.yaml, /cr/README.md
  - **Tests to add/update**: CR validation tests
  - **Exit criteria**:
    - Valid Agent CR with auto-prepare capability
    - Proper namespace and metadata structure
    - `kubectl --dry-run=client apply` succeeds for all examples

- [ ] **Controller skeleton** (ID: controller-skeleton)
  - **Goal**: Create basic controller structure with AWS client interface
  - **Changed files**: /controller/main.go, /controller/reconcile.go, /controller/bedrock_client.go
  - **Tests to add/update**: Controller initialization tests, reconcile logic tests
  - **Exit criteria**:
    - Main controller with proper signal handling and graceful shutdown
    - Reconcile function with idempotent logic structure
    - Bedrock client interface (no real AWS calls yet)
    - Informer and workqueue setup using client-go patterns
    - `make build-local` succeeds

- [ ] **Unit tests** (ID: unit-tests)
  - **Goal**: Comprehensive unit tests with mocked AWS client
  - **Changed files**: /controller/*_test.go
  - **Tests to add/update**: All reconcile logic, client interface, type conversions
  - **Exit criteria**:
    - Mock AWS client interface for offline testing
    - Reconcile logic tests with fake informer cache
    - Test coverage >80% for controller package
    - `make test` passes with race detection

- [ ] **Complete Makefile targets** (ID: makefile-targets)
  - **Goal**: Ensure all Makefile targets work correctly
  - **Changed files**: Makefile (refinements)
  - **Tests to add/update**: Makefile target validation
  - **Exit criteria**:
    - All build targets (local, deploy, docker) work
    - Test and lint targets execute successfully
    - CRD validation and sync check targets functional

- [ ] **Run local tests** (ID: run-local-tests)
  - **Goal**: Verify all tests pass and fix any failures
  - **Changed files**: Fix any test failures across codebase
  - **Tests to add/update**: Address test failures and gaps
  - **Exit criteria**:
    - `make test` passes completely
    - `make lint` passes with no issues
    - `make dev-check` succeeds end-to-end

## Future Sprints (Planned)

### Sprint 2: Agent Implementation
- [ ] Implement Agent CRUD operations with AWS Bedrock Agent service
- [ ] Add Agent preparation functionality with auto-prepare option
- [ ] Agent status reporting and condition management
- [ ] Integration tests with mocked AWS responses

### Sprint 3: Inference Profiles
- [ ] Implement Inference Profile CRUD operations
- [ ] Integration with Bedrock runtime service
- [ ] Inference Profile status and metrics

### Sprint 4: Knowledge Bases  
- [ ] Implement Knowledge Base CRUD operations
- [ ] Vector database integration
- [ ] Data source configuration

### Sprint 5: Advanced Agent Features
- [ ] Action Groups support
- [ ] Flows integration
- [ ] Guardrails implementation
- [ ] Aliases management
- [ ] Multi-Agent Collaboration
- [ ] Versions and deployment management

## Notes

- Following iterative development: complete one thin slice at a time
- All unit tests must pass before moving to next task
- CRD, types, examples, and controller code must stay in lockstep
- Using client-go patterns (no Kubebuilder/Operator SDK)
- AWS SDK Go v2 with pinned versions
- Offline unit tests only (no network calls in tests)
