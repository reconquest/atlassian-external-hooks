var Context = function () {
    this.state = require('bitbucket/util/state')

    this.getProjectKey = function() {
        return this.state.getProject().key
    }

    this.getRepositorySlug = function() {
        return this.state.getRepository().slug
    }

    this.getPullRequestID = function() {
        return this.state.getPullRequest().id
    }

    return this;
}
