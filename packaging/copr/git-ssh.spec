%global debug_package %{nil}

Name:           git-ssh
Version:        0.1.0
Release:        1%{?dist}
Summary:        Per-repo SSH identity profiles for GitHub (Orca-safe)

License:        MIT
URL:            https://github.com/kuyacarlo/ssh-profile
Source0:        %{url}/archive/refs/tags/v%{version}.tar.gz#/%{name}-%{version}.tar.gz

BuildRequires:  golang

%description
git-ssh manages per-repo SSH identity profiles for GitHub without relying
on ~/.ssh/config Host aliases, so it stays compatible with tools (like
Orca) that need a stable, unaliased origin URL.

%prep
%autosetup -n ssh-profile-%{version}

%build
export CGO_ENABLED=0
export GOTOOLCHAIN=auto
go build -ldflags "-s -w -X main.Version=%{version} -X main.CommitHash=rpm -X main.CompileDate=%(date +%%FT%%T%%z)" \
    -o git-ssh .

%install
install -Dm0755 git-ssh %{buildroot}%{_bindir}/git-ssh

%files
%{_bindir}/git-ssh
%license LICENSE

%changelog
* Mon Jul 27 2026 kuyacarlo <106532351+kuyacarlo@users.noreply.github.com> - 0.1.0-1
- Initial packaging
