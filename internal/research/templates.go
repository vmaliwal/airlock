package research

const ResearchContractTemplate = `{
  "objective": "Describe the exact issue and success condition",
  "mode": "mutate",
  "airlock": {
    "backend": { "kind": "lima" },
    "sandbox": {
      "namePrefix": "repo-issue-name",
      "artifactsDir": "/tmp/airlock-run-artifacts",
      "cpu": 4,
      "memoryGiB": 8,
      "diskGiB": 20,
      "ttlMinutes": 60
    },
    "repo": {
      "cloneUrl": "https://github.com/org/repo.git",
      "ref": "main"
    },
    "security": {
      "bootstrapNetwork": "allowlist",
      "bootstrapAllowHosts": ["archive.ubuntu.com", "security.ubuntu.com", "ports.ubuntu.com"],
      "bootstrapAptPackages": ["git", "ca-certificates"],
      "network": "allowlist",
      "allowHosts": ["github.com"],
      "allowedEnv": [],
      "exportPaths": ["/airlock/artifacts"],
      "includePatch": true
    },
    "steps": [
      { "name": "placeholder", "run": "true" }
    ]
  },
  "setup": [],
  "reproduction": {
    "command": "repo-native failing command",
    "repeat": 1,
    "success": { "min_failures": 1 }
  },
  "patches": [
    {
      "name": "minimal bounded fix",
      "command": "repo-native patch command"
    }
  ],
  "validation": {
    "target_command": "same or stronger target command",
    "repeat": 1,
    "success": { "exit_code": 0, "min_pass_rate": 1.0, "max_failures": 0 }
  },
  "safety": {
    "max_files_changed": 2,
    "max_loc_changed": 40,
    "allowed_paths": ["path/to/allowed/*"]
  },
  "stuck": {
    "max_same_failure_fingerprint": 2,
    "max_attempts_without_improvement": 2
  }
}
`

const CampaignPlanTemplate = `{
  "objective": "Run multiple verified issue contracts as one aggregate campaign",
  "artifactsDir": "/tmp/airlock-campaign-artifacts",
  "entries": [
    {
      "name": "issue-one",
      "contract": "issue-one-research.json"
    },
    {
      "name": "issue-two",
      "contract": "issue-two-research.json"
    }
  ],
  "success": {
    "max_failed": 0
  }
}
`

const AttemptTemplate = `{
  "repo": "/absolute/path/to/git/repo",
  "artifactsDir": "/tmp/airlock-attempt-artifacts",
  "attempt": {
    "name": "bounded-fix-name",
    "commit_message": "attempt: bounded fix",
    "validation": {
      "command": "repo-native validation command",
      "repeat": 1,
      "success": { "exit_code": 0, "min_pass_rate": 1.0 }
    },
    "safety": {
      "max_files_changed": 1,
      "allowed_paths": ["path/to/file"]
    }
  },
  "mutation": {
    "search_replace": {
      "path": "path/to/file",
      "oldText": "before",
      "newText": "after"
    }
  }
}
`

const AutofixTemplate = `{
  "objective": "Given a bug, try bounded fixes until one validates",
  "repo": "/absolute/path/to/git/repo",
  "artifactsDir": "/tmp/airlock-autofix-artifacts",
  "fingerprint_hints": ["package_failure:example/module"],
  "attempts": [
    {
      "attempt": {
        "name": "candidate-one",
        "validation": {
          "command": "repo-native validation command",
          "repeat": 1,
          "success": { "exit_code": 0, "min_pass_rate": 1.0 }
        },
        "safety": {
          "max_files_changed": 1,
          "allowed_paths": ["path/to/file"]
        }
      },
      "mutation": {
        "search_replace": {
          "path": "path/to/file",
          "oldText": "before",
          "newText": "after"
        }
      }
    },
    {
      "attempt": {
        "name": "candidate-two",
        "validation": {
          "command": "repo-native validation command",
          "repeat": 1,
          "success": { "exit_code": 0, "min_pass_rate": 1.0 }
        },
        "safety": {
          "max_files_changed": 1,
          "allowed_paths": ["path/to/file"]
        }
      },
      "mutation": {
        "replace_line": {
          "path": "path/to/file",
          "oldLine": "before line",
          "newLine": "after line"
        }
      }
    }
  ]
}
`
