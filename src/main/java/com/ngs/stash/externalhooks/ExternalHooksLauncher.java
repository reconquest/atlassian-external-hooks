package com.ngs.stash.externalhooks;

import java.util.EnumSet;
import java.util.Set;

import javax.annotation.concurrent.GuardedBy;

import com.atlassian.event.api.EventListener;
import com.atlassian.event.api.EventPublisher;
import com.atlassian.plugin.event.events.PluginEnabledEvent;
import com.atlassian.plugin.event.events.PluginModuleEnabledEvent;
import com.atlassian.plugin.spring.scanner.annotation.imports.ComponentImport;
import com.atlassian.sal.api.lifecycle.LifecycleAware;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.DisposableBean;
import org.springframework.beans.factory.InitializingBean;

public class ExternalHooksLauncher implements LifecycleAware, InitializingBean, DisposableBean {
  private static final Logger logger = LoggerFactory.getLogger(ExternalHooksLauncher.class);

  private final EventPublisher eventPublisher;
  private final ExternalHooksService service;

  @GuardedBy("this")
  private final Set<LifecycleEvent> lifecycleEvents = EnumSet.noneOf(LifecycleEvent.class);

  public ExternalHooksLauncher(
      @ComponentImport EventPublisher eventPublisher,
      @ComponentImport ExternalHooksService snakeService) {
    this.eventPublisher = eventPublisher;

    this.service = snakeService;
  }

  /**
   * This is received from Spring after the bean's properties are set. We need to accept this to
   * know when it is safe to register an event listener.
   */
  @Override
  public void afterPropertiesSet() {
    registerListener();
    onLifecycleEvent(LifecycleEvent.AFTER_PROPERTIES_SET);
  }

  /**
   * This is received from SAL after the system is really up and running from its perspective. This
   * includes things like the database being set up and other tricky things like that. This needs to
   * happen before we try to schedule anything, or the scheduler's tables may not be in a good state
   * on a clean install.
   */
  @Override
  public void onStart() {
    onLifecycleEvent(LifecycleEvent.LIFECYCLE_AWARE_ON_START);
  }

  @Override
  public void onStop() {
    //
  }

  /**
   * This is received from the plugin system after the plugin is fully initialized. It is not safe
   * to use Active Objects before this event is received.
   */
  @EventListener
  public void onPluginEnabled(PluginEnabledEvent event) {
    String pluginKey = event.getPlugin().getKey();
    if (Const.PLUGIN_KEY.equals(pluginKey)) {
      onLifecycleEvent(LifecycleEvent.PLUGIN_ENABLED);
    }
  }

  @EventListener
  public void onPluginModuleEnabled(PluginModuleEnabledEvent event) {
    // nope
  }

  /**
   * This is received from Spring when we are getting destroyed. We should make sure we do not leave
   * any event listeners or job runners behind; otherwise, we could leak the current plugin context,
   * leading to exceptions from destroyed OSGi proxies, memory leaks, and strange behaviour in
   * general.
   */
  @Override
  public void destroy() throws Exception {
    unregisterListener();
  }

  /**
   * The latch which ensures all of the plugin/application lifecycle progress is completed before we
   * call {@code launch()}.
   */
  private void onLifecycleEvent(LifecycleEvent event) {
    if (logger.isInfoEnabled()) {
      logger.info("onLifecycleEvent: " + event);
    }
    if (isLifecycleReady(event)) {
      unregisterListener();
      try {
        launch();
      } catch (Exception ex) {
        logger.error("Unexpected error during launch", ex);
      }
    }
  }

  /**
   * The event latch.
   *
   * <p>When something related to the plugin initialization happens, we call this with the
   * corresponding type of the event. We will return {@code true} at most once, when the very last
   * type of event is triggered. This method has to be {@code synchronized} because {@code EnumSet}
   * is not thread-safe and because we have multiple accesses to {@code lifecycleEvents} that need
   * to happen atomically for correct behaviour.
   *
   * @param event the lifecycle event that occurred
   * @return {@code true} if this completes the set of initialization-related events; {@code false}
   *     otherwise
   */
  private synchronized boolean isLifecycleReady(LifecycleEvent event) {
    return lifecycleEvents.add(event) && lifecycleEvents.size() == LifecycleEvent.values().length;
  }

  /** Do all the things we can't do before the system is fully up. */
  private void launch() throws Exception {
    initActiveObjects();

    service.start();
  }

  private void registerListener() {
    logger.debug("register initialisation controller");
    eventPublisher.register(this);
  }

  private void unregisterListener() {
    logger.debug("unregister initialisation controller");
    eventPublisher.unregister(this);
  }

  /**
   * Prod AO to make sure it is really and truly ready to go. If AO needs to do things like upgrade
   * the schema or if it is going to completely blow up on us, then hopefully that will happen here.
   * If we don't do this, then AO will do all of these things when we first touch it at some
   * arbitrary other point in the code, meaning that the place where the upgrades, failures, etc.
   * happen might not be deterministic. Explicitly prodding AO here makes the system more
   * deterministic and therefore easier to troubleshooting.
   */
  private void initActiveObjects() {
    logger.debug("initialise ActiveObjects");
  }

  /**
   * Used to keep track of everything that needs to happen before we are sure that it is safe to
   * talk to all of the components we need to use, particularly the {@code SchedulerService} and
   * Active Objects. We will not try to initialize until all of them have happened.
   */
  static enum LifecycleEvent {
    AFTER_PROPERTIES_SET,
    PLUGIN_ENABLED,
    LIFECYCLE_AWARE_ON_START
  }
}
