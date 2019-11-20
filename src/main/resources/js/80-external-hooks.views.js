var ViewGlobalSettings = function (context, api) {
    this._$ = $('#rq_hooks_global_settings_form');
    if (this._$.length == 0) {
        return ViewNotApplicable;
    }

    this._$spinner = new Spinner();
    this._$progress = new ProgressBarWithText();

    this.mount = function () {
        this._$.find('h2').append(this._$spinner);

        this._$.submit(function (e) {
            e.preventDefault();

            this._setLoading(true);

            var updating = this._updateSettings();

            if (this._$.find('[name="apply-existing"]').prop('checked')) {
                updating
                    .then(this._applySettings.bind(this))
                    .then(this._renderApplyProgress.bind(this))
                    .then(this._monitorApplyProgress.bind(this))
            }

            updating.done(this._setLoading.bind(this, false));
        }.bind(this));

        this._setLoading(true);
        this._loadSettings()
            .done(this._setLoading.bind(this, false));
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
            500
        );

        return promise;
    }

    this._renderApplyProgress = function (state) {
        if (state.started) {
            this._$progress
                .setIndeterminate(false)
                .setTotal(state.total)
                .setCurrent(state.current)

            if (state.finished) {
                this._$progress.setText(
                    state.total
                        + " hook" + (state.total > 1 ? "s were" : " was")
                        + " updated."
                )
            } else {
                this._$progress.setText(
                    "Configuring hook "
                        + state.current + " of " + state.total + "…"
                );
            }
        } else {
            this._$progress
                .setText("Initializing…");
        }

        return state;
    }

    this._loadSettings = function () {
        this._$.find('input').prop('checked', false);

        return api.getSettings()
            .done(
                function (settings) {
                    $.each(settings.triggers, function(hook, events) {
                        $.each(events, function (_, event) {
                            var name = 'triggers.' + hook + '.' + event;

                            this._$
                                .find('[name="' + name + '"]')
                                .prop('checked', true);
                        }.bind(this))
                    }.bind(this))
                }.bind(this)
            );
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
