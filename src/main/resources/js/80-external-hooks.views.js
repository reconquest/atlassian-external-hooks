var ViewGlobalSettings = function (context, api) {
    this._$ = $('#rq_hooks_global_settings_form');
    if (this._$.length == 0) {
        return ViewNotApplicable;
    }

    this._$spinner = new Spinner();

    this.mount = function () {
        this._$.find('h2').append(this._$spinner);

        this._$.submit(function (e) {
            e.preventDefault();

            var settings = this._getSettings();

            console.log(settings);
        }.bind(this));

        this._loadSettings();
    }

    this._loadSettings = function () {
        this._$spinner.show();

        api.getSettings()
            .done(function (settings) {
                this._$spinner.hide();

                $.each(settings.triggers, function(hook, events) {
                    $.each(events, function (_, event) {
                        var name = 'triggers.' + hook + '.' + event;

                        this._$
                            .find('[name="' + name + '"]')
                            .prop('checked', true);
                    }.bind(this))
                }.bind(this))
            }.bind(this));
    }

    this._getSettings = function() {
        var settings = {};

        this._$.find('[name^="triggers."]').each(function () {
            var matches = $(this).attr('name').match(/triggers\.(\S+)\.(\S+)/);

            if (!settings[matches[1]]) {
                settings[matches[1]] = {};
            }

            settings[matches[1]][matches[2]] = $(this).prop('checked');
        });

        return settings;
    }

    return this;
}

var views = [
    ViewGlobalSettings
];
