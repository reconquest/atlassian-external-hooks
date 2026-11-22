package com.ngs.stash.externalhooks;

import static org.mockito.Mockito.inOrder;
import static org.mockito.Mockito.mock;

import com.atlassian.event.api.EventPublisher;

import org.junit.Test;
import org.mockito.InOrder;

public class BitbucketEventListenerTest {
  @Test
  public void eventSubscriptionFollowsBeanLifecycle() {
    EventPublisher publisher = mock(EventPublisher.class);
    BitbucketEventListener listener = new BitbucketEventListener(publisher, null, null, null);

    listener.afterPropertiesSet();
    listener.destroy();

    InOrder calls = inOrder(publisher);
    calls.verify(publisher).register(listener);
    calls.verify(publisher).unregister(listener);
    calls.verifyNoMoreInteractions();
  }
}
