rm clone-apps-plugin
go build
cf install-plugin clone-apps-plugin -f
