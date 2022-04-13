package main

import (
	"dagger.io/dagger"
	"dagger.io/dagger/core"
)

dagger.#Plan & {
	actions: {
		image: core.#Pull & {
			source: "alpine:3.15.0@sha256:e7d88de73db3d3fd9b2d63aa7f447a10fd0220b7cbf39803c803f2af9ba256b3"
		}

		addUser: core.#Exec & {
			input: image.output
			args: ["adduser", "-D", "test"]
		}
		test: {
			verifyUsername: core.#Exec & {
				input: addUser.output
				user:  "test"
				args: [
					"sh", "-c",
					"""
						test "$(whoami)" = "test"
						""",
				]
			}

			verifyUserID: core.#Exec & {
				input: addUser.output
				user:  "1000"
				args: [
					"sh", "-c",
					"""
						test "$(whoami)" = "test"
						""",
				]
			}
		}
	}
}
