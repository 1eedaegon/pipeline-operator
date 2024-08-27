/* tslint:disable */
/* eslint-disable */
/**
 * Kubernetes
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * The version of the OpenAPI document: v1.30.0
 * 
 *
 * NOTE: This class is auto generated by OpenAPI Generator (https://openapi-generator.tech).
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */

import { mapValues } from '../runtime';
import type { IoK8sApimachineryPkgApisMetaV1ListMeta } from './IoK8sApimachineryPkgApisMetaV1ListMeta';
import {
    IoK8sApimachineryPkgApisMetaV1ListMetaFromJSON,
    IoK8sApimachineryPkgApisMetaV1ListMetaFromJSONTyped,
    IoK8sApimachineryPkgApisMetaV1ListMetaToJSON,
} from './IoK8sApimachineryPkgApisMetaV1ListMeta';
import type { IoGithub1eedaegonPipelineV1Pipeline } from './IoGithub1eedaegonPipelineV1Pipeline';
import {
    IoGithub1eedaegonPipelineV1PipelineFromJSON,
    IoGithub1eedaegonPipelineV1PipelineFromJSONTyped,
    IoGithub1eedaegonPipelineV1PipelineToJSON,
} from './IoGithub1eedaegonPipelineV1Pipeline';

/**
 * PipelineList is a list of Pipeline
 * @export
 * @interface IoGithub1eedaegonPipelineV1PipelineList
 */
export interface IoGithub1eedaegonPipelineV1PipelineList {
    /**
     * APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources
     * @type {string}
     * @memberof IoGithub1eedaegonPipelineV1PipelineList
     */
    apiVersion?: string;
    /**
     * List of pipelines. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md
     * @type {Array<IoGithub1eedaegonPipelineV1Pipeline>}
     * @memberof IoGithub1eedaegonPipelineV1PipelineList
     */
    items: Array<IoGithub1eedaegonPipelineV1Pipeline>;
    /**
     * Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
     * @type {string}
     * @memberof IoGithub1eedaegonPipelineV1PipelineList
     */
    kind?: string;
    /**
     * 
     * @type {IoK8sApimachineryPkgApisMetaV1ListMeta}
     * @memberof IoGithub1eedaegonPipelineV1PipelineList
     */
    metadata?: IoK8sApimachineryPkgApisMetaV1ListMeta;
}

/**
 * Check if a given object implements the IoGithub1eedaegonPipelineV1PipelineList interface.
 */
export function instanceOfIoGithub1eedaegonPipelineV1PipelineList(value: object): value is IoGithub1eedaegonPipelineV1PipelineList {
    if (!('items' in value) || value['items'] === undefined) return false;
    return true;
}

export function IoGithub1eedaegonPipelineV1PipelineListFromJSON(json: any): IoGithub1eedaegonPipelineV1PipelineList {
    return IoGithub1eedaegonPipelineV1PipelineListFromJSONTyped(json, false);
}

export function IoGithub1eedaegonPipelineV1PipelineListFromJSONTyped(json: any, ignoreDiscriminator: boolean): IoGithub1eedaegonPipelineV1PipelineList {
    if (json == null) {
        return json;
    }
    return {
        
        'apiVersion': json['apiVersion'] == null ? undefined : json['apiVersion'],
        'items': ((json['items'] as Array<any>).map(IoGithub1eedaegonPipelineV1PipelineFromJSON)),
        'kind': json['kind'] == null ? undefined : json['kind'],
        'metadata': json['metadata'] == null ? undefined : IoK8sApimachineryPkgApisMetaV1ListMetaFromJSON(json['metadata']),
    };
}

export function IoGithub1eedaegonPipelineV1PipelineListToJSON(value?: IoGithub1eedaegonPipelineV1PipelineList | null): any {
    if (value == null) {
        return value;
    }
    return {
        
        'apiVersion': value['apiVersion'],
        'items': ((value['items'] as Array<any>).map(IoGithub1eedaegonPipelineV1PipelineToJSON)),
        'kind': value['kind'],
        'metadata': IoK8sApimachineryPkgApisMetaV1ListMetaToJSON(value['metadata']),
    };
}

