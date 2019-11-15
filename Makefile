run@%:
	@mkdir -p targets/$*
	@ln -sTf targets/$* target
	@atlas-run -Dbitbucket.version=$*

package:
	@atlas-mvn package -q -T $(shell nproc)
