var ViewGlobalHooks = function (context, api) {
    this._$ = $('#rq_hooks_global_hooks_form');
    if (this._$.length == 0) {
        return new ViewNotApplicable();
    }

    this._$spinner = new Spinner();

    this.mount = function () {
        this._$.find('h2').append(this._$spinner);

        this._$.find('[data-rq-hook-kind]').each(function () {
            var $pane = $(this);
            var kind = $pane.attr('data-rq-hook-kind');

            $pane.find('[name$=".enabled"]').change(function () {
                $pane.find('.rq-global-hook-settings').
                    toggleClass('rq-hook-disabled', !$(this).prop('checked'));
            })
        });

        this._$.submit(function (e) {
            e.preventDefault();

            this._withLoader(this._saveSettings.bind(this));
        }.bind(this));

        this._withLoader(this._loadSettings.bind(this));
    }

    this._withLoader = function (fn) {
        this._setLoading(true);
        fn().then(this._setLoading.bind(this, false));
    }

    this._setLoading = function (loading) {
        if (loading) {
            this._$.find('input, button, textarea').prop('disabled', true);
            this._$spinner.show();
        } else {
            this._$.find('input, button, textarea').prop('disabled', false);
            this._$spinner.hide();
        }
    }

    this._saveSettings = function () {
        var updating = this._updateSettings();

        if (this._$.find('[name="apply-existing"]').prop('checked')) {
            updating = updating
                .then(this._applySettings.bind(this))
        }

        var flag = new FlagSuccess('Settings successfully updated.');

        return updating.then(flag.show.bind(flag));
    }

    this._updateSettings = function () {
        return api.updateSettings(this._getSettings());
    }

    this._applySettings = function () {
        return api.runHooksFactory();
    }

    this._renderSettings = function (kind, settings) {
        console.log(kind, settings);
        //$.each(settings.triggers, function(hook, events) {
        //    $.each(events, function (_, event) {
        //        var name = 'triggers.' + hook + '.' + event;

        //        this._$
        //            .find('[name="' + name + '"]')
        //            .prop('checked', true);
        //    }.bind(this))
        //}.bind(this))
    }

    this._loadSettings = function () {
        var kinds = this._$.find('[data-rq-hook-kind]').
            map(function (_, el) { return el.getAttribute('data-rq-hook-kind'); }).
            get();

        return $.when.apply(
            $,
            kinds.map(function(kind) {
                return api.getGlobalHook(kind).then(function (settings) {
                    return {
                        kind: kind,
                        settings: settings
                    };
                })
            })
        ).done(function() {
            console.log(arguments);
            $.each(arguments, function (_, hook) {
                this._renderSettings(hook.kind, hook.settings);
            }.bind(this))
        }.bind(this));

        //return api.getSettings()
        //    .done(this._renderSettings.bind(this));
    }

    this._loadSettingsDefaults = function () {
        return api.getSettingsDefaults()
            .done(this._renderSettings.bind(this));
    }

    this._getSettings = function() {
        var triggers = {};

        this._$.find('[name^="triggers."]')
            .each(
                function () {
                    var matches = $(this).attr('name')
                        .match(/triggers\.(\S+)\.(\S+)/);

                    if (!$(this).prop('checked')) {
                        return;
                    }

                    if (!triggers[matches[1]]) {
                        triggers[matches[1]] = [];
                    }

                    triggers[matches[1]].push(matches[2]);
                }
            );

        return {
            'triggers': triggers
        }
    }

    return this;
}
