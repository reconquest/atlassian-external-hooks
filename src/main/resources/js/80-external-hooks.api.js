var API = function (baseURL) {
    this.urls = Object.create({
        root: function() {
            return baseURL + '/rest/external-hooks/1.0/';
        },

        settings: function () {
            return this.root() + 'settings';
        }
    });

    this._headers = {
        "X-Atlassian-Token": "no-check"
    };

    this.getSettings = function () {
        return $.ajax(
            this.urls.settings(),
            {
                method: "GET",
                headers: this._headers
            }
        );
    }

    return this;
}
