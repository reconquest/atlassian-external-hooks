var API = function (baseURL) {
    this.urls = Object.create({
        root: function() {
            return baseURL + '/rest/external-hooks/1.0/';
        },

        settings: function () {
            return this.root() + 'settings';
        },

        settingsDefaults: function () {
            return this.root() + 'settings/default';
        },

        factory: function () {
            return this.root() + 'factory/';
        },

        factoryHooks: function () {
            return this.factory() + 'hooks';
        },

        factoryState: function (id) {
            return this.factory() + 'state/' + id;
        },

        globalHook: function(kind) {
            return this.root() + '/global-hooks/' + kind;
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

    this.getSettingsDefaults = function () {
        return $.ajax(
            this.urls.settingsDefaults(),
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

    this.getGlobalHook = function (kind) {
        return $.ajax(
            this.urls.globalHook(kind),
            {
                method: "GET",
                headers: this._headers
            }
        );
    }

    this.setGlobalHook = function (kind, settings) {
        return $.ajax(
            this.urls.globalHook(kind),
            {
                method: "PUT",
                headers: this._headers,
                data: JSON.stringify(settings),
                contentType: "application/json; charset=UTF-8"
            }
        );
    }

    return this;
}
