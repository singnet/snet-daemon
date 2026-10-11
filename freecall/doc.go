// Package freecall authorizes token-based free calls and tracks per-user usage.
// It shares the payment interceptor and atomic storage with escrow, without
// depending on payment channels.
package freecall
