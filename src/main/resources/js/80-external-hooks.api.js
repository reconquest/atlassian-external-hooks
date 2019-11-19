var API = function (baseURL) {
    this.urls = Object.create({
        root: function() {
            return baseURL + '/rest/external-hooks/1.0/';
        },

        settings: function () {
            return this.root() + 'settings';
        },

        factory: function () {
            return this.root() + 'factory/';
        },

        factoryHooks: function () {
            return this.factory() + 'hooks';
        },

        factoryState: function (id) {
            return this.factory() + 'state/' + id;
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

    this.updateSettings = function (settings) {
        return $.ajax(
            this.urls.settings(),
            {
                data: JSON.stringify(settings),
                method: "PUT",
                headers: this._headers,
                contentType: "application/json; charset=UTF-8"
            }
        );
    }

    this.runHooksFactory = function () {
        return $.ajax(
            this.urls.factoryHooks(),
            {
                method: "POST",
                headers: this._headers
            }
        );
    }

    this.getHooksFactoryState = function (id) {
        return $.ajax(
            this.urls.factoryState(id),
            {
                method: "GET",
                headers: this._headers
            }
        );
    }

    return this;
}
