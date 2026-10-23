// Package freecall authorizes token-based free calls and tracks per-user usage.
// It shares the payment interceptor and atomic storage with escrow, without
// depending on payment channels.
//
// Registry supplies the URIs of organization and service metadata. The daemon
// loads that metadata off chain to select the replica group, token-signing
// address, and default quota. Registry does not validate or enforce free-call
// counters or quota values. The free-call signer is distinct in purpose from
// Registry ownership and the MPE channel sender, signer, or recipient.
//
// Token issuance for ordinary wallets reads their configured network ERC-20
// balance, not an MPE deposit or channel balance. Tokens and usage updates do not
// submit transactions to MPE or Registry. Quotas are enforced by the daemon using
// its configured storage backend and local per-address overrides.
//
// Caller signatures bind the organization, service, and metadata group ID.
// Provider-issued tokens instead bind the organization, daemon group name,
// caller address, optional user ID, and expiration block. Their payload contains
// neither a service ID nor an MPE contract address. The historical escrow protobuf
// namespace is retained for client compatibility, not a payment-channel dependency.
package freecall
