run@%:
	@mkdir -p target-$*
	@ln -sTf target-$* target
	@atlas-run -Dbitbucket.version=$*

package:
	@atlas-mvn package -q -T $(shell nproc)
