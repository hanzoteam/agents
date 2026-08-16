// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {BaseWebSocketMessage} from '@hanzoteam/client';

import {PostUpdateWebsocketMessage} from './components/llmbot_post/llmbot_post';

type WebsocketListener = (msg: BaseWebSocketMessage<string, PostUpdateWebsocketMessage>) => void
type WebsocketListenerObject = {
    postID: string;
    listenerID: string;
    listener: WebsocketListener;
}
type WebsocketListeners = WebsocketListenerObject[]

export default class PostEventListener {
    postUpdateWebsocketListeners: WebsocketListeners = [];

    public registerPostUpdateListener = (postID: string, listenerID: string, listener: WebsocketListener) => {
        this.postUpdateWebsocketListeners.push({postID, listenerID, listener});
    };

    public unregisterPostUpdateListener = (postID: string, listenerID: string) => {
        this.postUpdateWebsocketListeners = this.postUpdateWebsocketListeners.filter((listenerObject) => {
            const isSamePostID = listenerObject.postID === postID;
            const isSameListenerID = listenerObject.listenerID === listenerID;
            return !(isSamePostID && isSameListenerID);
        });
    };

    public handlePostUpdateWebsockets = (msg: BaseWebSocketMessage<string, PostUpdateWebsocketMessage>) => {
        const postID = msg.data.post_id;
        this.postUpdateWebsocketListeners.forEach((listenerObject) => {
            if (listenerObject.postID === postID) {
                listenerObject.listener(msg);
            }
        });
    };
}
