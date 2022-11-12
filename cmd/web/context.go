package main

type contextKey string

var isAuthenticatedContextKey = contextKey("isAuthenticated")
var numberOfCardsContextKey = contextKey("numberOfCards")
