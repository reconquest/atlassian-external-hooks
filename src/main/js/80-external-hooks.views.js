var ViewGlobalSettings = function (context, api) {
    this._$ = $('#rq_hooks_global_settings_form');
    if (this._$.length == 0) {
        return new ViewNotApplicable();
    }

    this._$spinner = new Spinner();
    this._$progress = new ProgressBarWithText();

    this.mount = function () {
        this._$.find('h2').append(this._$spinner);

        this._$.submit(function (e) {
            e.preventDefault();

            this._withLoader(this._saveSettings.bind(this));
        }.bind(this));

        this._$.find('#rq_hooks_settings_defaults').click(
            this._withLoader.bind(this, this._loadSettingsDefaults.bind(this))
        )

        this._withLoader(this._loadSettings.bind(this));
    }

    this._withLoader = function (fn) {
        this._setLoading(true);
        fn().then(this._setLoading.bind(this, false));
    }

    this._setLoading = function (loading) {
        if (loading) {
            this._$.find('input, button').prop('disabled', true);
            this._$spinner.show();
        } else {
            this._$.find('input, button').prop('disabled', false);
            this._$spinner.hide();
        }
    }

    this._saveSettings = function () {
        var updating = this._updateSettings();

        if (this._$.find('[name="apply-existing"]').prop('checked')) {
            updating = updating
                .then(this._applySettings.bind(this))
                .then(this._renderApplyProgress.bind(this))
                .then(this._monitorApplyProgress.bind(this))
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

    this._monitorApplyProgress = function (state) {
        this._$progress.setIndeterminate(true);
        this._$.append(this._$progress);

        var promise = $.Deferred();

        var monitor = setInterval(
            function () {
                api.getHooksFactoryState(state.id)
                    .done(
                        function (state) {
                            this._renderApplyProgress(state);

                            if (state.finished) {
                                clearInterval(monitor);
                                promise.resolve();
                            }
                        }.bind(this)
                    );
            }.bind(this),
            200
        );

        return promise.promise();
    }

    this._renderApplyProgress = function (state) {
        if (state.started && (state.total > 0 || state.finished)) {
            this._$progress
                .setIndeterminate(false)
                .setCurrent(state.current)
                .setTotal(state.total)

            if (state.finished) {
                if (state.total == 0) {
                    this._$progress.setText("No existing hooks to update.");
                } else {
                    this._$progress.setText(
                        state.total
                        + " hook" + (state.total > 1 ? "s were" : " was")
                        + " updated."
                    )
                }
            } else {
                this._$progress.setText(
                    "Configuring hook "
                        + state.current + " of " + state.total + "…"
                );
            }
        } else {
            this._$progress
                .setIndeterminate(true)
                .setText("Initializing…");
        }

        return state;
    }

    this._renderSettings = function (settings) {
        this._$.find('input').prop('checked', false);

        $.each(settings.triggers, function(hook, events) {
            $.each(events, function (_, event) {
                var name = 'triggers.' + hook + '.' + event;

                this._$
                    .find('[name="' + name + '"]')
                    .prop('checked', true);
            }.bind(this))
        }.bind(this))
    }

    this._loadSettings = function () {
        return api.getSettings()
            .done(this._renderSettings.bind(this));
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

var views = [
    ViewGlobalSettings
];
