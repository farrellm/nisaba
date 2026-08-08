package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"

	"github.com/farrellm/nisaba/internal/config"
	"github.com/farrellm/nisaba/internal/db"
	"github.com/farrellm/nisaba/internal/store"
)

const minPasswordLen = 8

// runUserCommand handles the -create-user/-list-users/-delete-user flags and
// exits; the server is never started. It deliberately opens only Postgres —
// unlike run() it skips the legacy reflex.db SQLite handle, which fails hard
// when the file is missing and has nothing to do with managing users.
func runUserCommand(ctx context.Context) error {
	cfg := config.Load()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database connect: %w", err)
	}
	defer pool.Close()

	st := store.New(pool)

	switch {
	case *listUsers:
		return listUsersCmd(ctx, st)
	case *createUser != "":
		return createUserCmd(ctx, st, *createUser)
	default:
		return deleteUserCmd(ctx, st, *deleteUser, *force)
	}
}

// listUsersCmd prints every user as a padded table.
func listUsersCmd(ctx context.Context, st *store.Store) error {
	users, err := st.ListUsers(ctx)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		fmt.Println("no users")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tUSERNAME\tCREATED\tSUBREDDIT\tSTREAMING")
	for _, u := range users {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%t\n",
			u.ID, u.Username, u.CreatedAt.Local().Format("2006-01-02 15:04"), u.Subreddit, u.StreamingEnabled)
	}
	return w.Flush()
}

// createUserCmd creates a user, prompting for the password so it never lands in
// shell history or the process table.
func createUserCmd(ctx context.Context, st *store.Store, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("username is required")
	}

	password, err := readPassword()
	if err != nil {
		return err
	}
	if len(password) < minPasswordLen {
		return fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user, err := st.CreateUser(ctx, username, string(hash))
	if errors.Is(err, store.ErrDuplicate) {
		return fmt.Errorf("username %q is already taken", username)
	}
	if err != nil {
		return err
	}

	fmt.Printf("Created user %s (id %d)\n", user.Username, user.ID)
	return nil
}

// deleteUserCmd removes a user. The DELETE cascades: documents and labels
// reference users(id) ON DELETE CASCADE, and everything under a document
// (blocks, attributes, responses, posts, label joins) cascades from there, so
// the single statement is complete. Sessions live only in signed cookies — a
// deleted user's stale cookie already fails the /api/auth/me lookup.
func deleteUserCmd(ctx context.Context, st *store.Store, username string, force bool) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("username is required")
	}

	user, err := st.GetUserByUsername(ctx, username)
	if errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("no such user %q", username)
	}
	if err != nil {
		return err
	}

	if !force {
		fmt.Printf("Delete user %s (id %d) and all their documents? [y/N] ", user.Username, user.ID)
		answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && answer == "" {
			// EOF with nothing typed: treat silence as "no".
			fmt.Println()
			return errors.New("aborted")
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "y", "yes":
		default:
			return errors.New("aborted")
		}
	}

	if err := st.DeleteUser(ctx, user.ID); err != nil {
		return err
	}

	fmt.Printf("Deleted user %s (id %d)\n", user.Username, user.ID)
	return nil
}

// readPassword reads a password from stdin. On a terminal it prompts twice with
// echo off and requires the two to match; when stdin is a pipe it reads a single
// line so scripts can supply the password non-interactively.
func readPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return "", fmt.Errorf("read password: %w", err)
		}
		return strings.TrimRight(line, "\r\n"), nil
	}

	fmt.Print("Password: ")
	first, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}

	fmt.Print("Confirm password: ")
	second, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}

	if string(first) != string(second) {
		return "", errors.New("passwords do not match")
	}
	return string(first), nil
}
