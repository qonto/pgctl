package pgctl

import (
	"fmt"
	"os"
)

func (a *App) CopySequences(sourceAlias, targetAlias string, runApply bool) {
	if !runApply {
		fmt.Println("🚧 DRY RUN MODE ACTIVATED 🚧")
	}

	sourceDB, ok := a.Config.Databases[sourceAlias]
	if !ok {
		fmt.Printf("❌ Source database %s does not exist. Add it to your configuration file and retry.\n", sourceDB.Database)
		os.Exit(1)
	}

	targetDB, ok := a.Config.Databases[targetAlias]
	if !ok {
		fmt.Printf("❌ Target database %s does not exist. Add it to your configuration file and retry.\n", targetDB.Database)
		os.Exit(1)
	}

	if runApply {
		fmt.Printf("| Copying sequences from source database %s in %s to target database %s in %s\n",
			sourceDB.Database, sourceAlias, targetDB.Database, targetAlias)
		err := sourceDB.CopySequences(targetDB)
		if err != nil {
			fmt.Printf("❌ Could not copy sequences from source database %s in %s to target database %s in %s: %v\n",
				sourceDB.Database, sourceAlias, targetDB.Database, targetAlias, err)
			os.Exit(1)
		}
		fmt.Printf("✅ Successfully copied sequences from source database %s in %s to target database %s in %s\n",
			sourceDB.Database, sourceAlias, targetDB.Database, targetAlias)
	} else {
		a.ListSequences(sourceAlias)
		fmt.Printf("👉 Sequences would be copied from source database %s in %s to target database %s in %s\n", sourceDB.Database, sourceAlias, targetDB.Database, targetAlias)
	}
}

func (a *App) CheckSequences(sourceAlias, targetAlias string) {
	sourceDB, ok := a.Config.Databases[sourceAlias]
	if !ok {
		fmt.Printf("❌ Source db %s does not exist. Add it to your configuration file and retry.\n", sourceAlias)
		os.Exit(1)
	}

	targetDB, ok := a.Config.Databases[targetAlias]
	if !ok {
		fmt.Printf("❌ Target db %s does not exist. Add it to your configuration file and retry.\n", targetAlias)
		os.Exit(1)
	}

	sourceSequences, err := sourceDB.ListSequences()
	if err != nil {
		fmt.Printf("❌ Unable to get sequences from source database: %s", err)
		os.Exit(1)
	}

	targetSequences, err := targetDB.ListSequences()
	if err != nil {
		fmt.Printf("❌ Unable to get sequences from target database: %s", err)
		os.Exit(1)
	}

	var sequencesAreDifferent bool
	for _, sourceSequence := range sourceSequences {
		sequenceFound := false
		for _, targetSequence := range targetSequences {
			if sourceSequence.FullName == targetSequence.FullName {
				sequenceFound = true
				if sourceSequence.LastValue != targetSequence.LastValue {
					fmt.Printf("Sequence %s has different last values: source %s %d and target %s %d\n",
						sourceSequence.FullName, sourceAlias, sourceSequence.LastValue, targetAlias, targetSequence.LastValue)
					sequencesAreDifferent = true
				} else {
					fmt.Printf("Sequence %s: source %s %d and target %s %d\n",
						sourceSequence.FullName, sourceAlias, sourceSequence.LastValue, targetAlias, targetSequence.LastValue)
				}
			}
		}
		if !sequenceFound {
			fmt.Printf("Sequence %s not found in target %s\n", sourceSequence.FullName, targetDB.Database)
			sequencesAreDifferent = true
		}
	}

	if sequencesAreDifferent {
		fmt.Printf("❌ Sequences are different in %s database %s and %s database %s\n",
			sourceAlias, sourceDB.Database, targetAlias, targetDB.Database)
		os.Exit(1)
	}
	if len(sourceSequences) == 0 {
		fmt.Printf("✅ No sequences found in source database %s\n", sourceDB.Database)
	}
	if len(targetSequences) == 0 {
		fmt.Printf("✅ No sequences found in target database %s\n", targetDB.Database)
	}

	fmt.Printf("✅ Sequences are the same in %s database %s and %s database %s\n",
		sourceAlias, sourceDB.Database, targetAlias, targetDB.Database)
}

func (a *App) ListSequences(alias string) {
	db, ok := a.Config.Databases[alias]
	if !ok {
		fmt.Printf("❌ Database alias %s does not exist. Add it to your configuration file and retry.\n", db.Database)
		os.Exit(1)
	}

	sequences, err := db.ListSequences()
	if err != nil {
		fmt.Printf("❌ Could not list sequences from database %s: %v\n", db.Database, err)
		os.Exit(1)
	}

	if len(sequences) == 0 {
		fmt.Printf("| No sequences found in database %s\n", db.Database)
		os.Exit(0)
	}

	fmt.Printf("| Found %d sequences in database %s:\n", len(sequences), db.Database)
	for _, seq := range sequences {
		fmt.Printf("|   %s (last_value: %d)\n", seq.FullName, seq.LastValue)
	}
}
