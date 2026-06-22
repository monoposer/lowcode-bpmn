// Package bpmn re-exports the Process Design domain (internal/domain/definition).
//
// Deprecated: import github.com/monoposer/lowcode-bpmn/internal/domain/definition directly.
package bpmn

import "github.com/monoposer/lowcode-bpmn/internal/domain/definition"

type (
	ElementKind        = definition.ElementKind
	ProcessDefinition  = definition.ProcessDefinition
	Element            = definition.Element
	SequenceFlow       = definition.SequenceFlow
	EventDefinitionType = definition.EventDefinitionType
	EventDefinition    = definition.EventDefinition
	ApprovalMode       = definition.ApprovalMode
	Registry           = definition.Registry
	Lane               = definition.Lane
	DataObject         = definition.DataObject
	DataStore          = definition.DataStore
	Pool               = definition.Pool
	MessageFlow        = definition.MessageFlow
	Collaboration      = definition.Collaboration
	MultiInstanceLoopCharacteristics = definition.MultiInstanceLoopCharacteristics
)

const (
	KindStartEvent          = definition.KindStartEvent
	KindEndEvent            = definition.KindEndEvent
	KindUserTask            = definition.KindUserTask
	KindScriptTask          = definition.KindScriptTask
	KindServiceTask         = definition.KindServiceTask
	KindSendTask            = definition.KindSendTask
	KindReceiveTask         = definition.KindReceiveTask
	KindBusinessRuleTask    = definition.KindBusinessRuleTask
	KindExclusiveGateway    = definition.KindExclusiveGateway
	KindParallelGateway     = definition.KindParallelGateway
	KindInclusiveGateway    = definition.KindInclusiveGateway
	KindSubProcess          = definition.KindSubProcess
	KindBoundaryEvent       = definition.KindBoundaryEvent
	KindIntermediateCatchEvent = definition.KindIntermediateCatchEvent
	KindIntermediateThrowEvent = definition.KindIntermediateThrowEvent
	KindEventBasedGateway   = definition.KindEventBasedGateway
	KindComplexGateway      = definition.KindComplexGateway
	KindCallActivity        = definition.KindCallActivity
	EventTypeNone           = definition.EventTypeNone
	EventTypeMessage        = definition.EventTypeMessage
	EventTypeSignal         = definition.EventTypeSignal
	EventTypeTimer          = definition.EventTypeTimer
	EventTypeConditional    = definition.EventTypeConditional
	EventTypeError          = definition.EventTypeError
	ApprovalAny             = definition.ApprovalAny
	ApprovalAll             = definition.ApprovalAll
	ApprovalSequential      = definition.ApprovalSequential
)

var AutomatedTaskKinds = definition.AutomatedTaskKinds
var ExtensionKinds = definition.ExtensionKinds

var (
	Validate                      = definition.Validate
	BuildRegistry                 = definition.BuildRegistry
	EvalCondition                 = definition.EvalCondition
	ParseApprovalMode             = definition.ParseApprovalMode
	RequiredApprovals             = definition.RequiredApprovals
	ValidateUserTaskApproval      = definition.ValidateUserTaskApproval
	ValidateTaskElement           = definition.ValidateTaskElement
	ValidateEventDefinition       = definition.ValidateEventDefinition
	IsAutomatedTask               = definition.IsAutomatedTask
	IsExtensionKind               = definition.IsExtensionKind
	BoundaryMessageMatch          = definition.BoundaryMessageMatch
	BoundarySignalMatch           = definition.BoundarySignalMatch
	BoundaryCancelsActivity       = definition.BoundaryCancelsActivity
	IsBoundaryHostKind            = definition.IsBoundaryHostKind
	ValidateCollaboration         = definition.ValidateCollaboration
	ValidateStartEventEventDefinition = definition.ValidateStartEventEventDefinition
	MessageStartMatch             = definition.MessageStartMatch
	SignalStartMatch              = definition.SignalStartMatch
	ConditionalStartMatch         = definition.ConditionalStartMatch
	BusinessKeyFromCorrelation    = definition.BusinessKeyFromCorrelation
	ResolveReturnTarget           = definition.ResolveReturnTarget
	ScopeElementIDs               = definition.ScopeElementIDs
	ScopeGatewayIDs               = definition.ScopeGatewayIDs
)
