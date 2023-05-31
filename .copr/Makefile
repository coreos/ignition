# Maintained in https://github.com/coreos/repo-templates
# Do not edit downstream.

.PHONY: srpm
srpm:
	dnf install -y git rpm-build rpmdevtools 
	# similar to https://github.com/actions/checkout/issues/760, but for COPR
	git config --global --add safe.directory '*'
	curl -LOf https://src.fedoraproject.org/rpms/ignition/raw/rawhide/f/ignition.spec
	version=$$(git describe --always --tags | sed -e 's,-,\.,g' -e 's,^v,,'); \
	git archive --format=tar --prefix=ignition-$$version/ HEAD | gzip > ignition-$$version.tar.gz; \
	sed -ie "s,^Version:.*,Version: $$version," ignition.spec
	sed -ie 's/^Patch/# Patch/g' ignition.spec  # we don't want any downstream patches
	spectool -g ignition.spec  # download any remaining sources (e.g. coreos-installer-dracut)
	rpmbuild -bs --define "_sourcedir ${PWD}" --define "_specdir ${PWD}" --define "_builddir ${PWD}" --define "_srcrpmdir ${PWD}" --define "_rpmdir ${PWD}" --define "_buildrootdir ${PWD}/.build" ignition.spec
	mv *.src.rpm $$outdir
