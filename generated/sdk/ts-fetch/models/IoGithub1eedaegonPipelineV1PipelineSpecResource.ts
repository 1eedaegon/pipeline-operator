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
import type { IoGithub1eedaegonPipelineV1PipelineSpecResourceGpu } from './IoGithub1eedaegonPipelineV1PipelineSpecResourceGpu';
import {
    IoGithub1eedaegonPipelineV1PipelineSpecResourceGpuFromJSON,
    IoGithub1eedaegonPipelineV1PipelineSpecResourceGpuFromJSONTyped,
    IoGithub1eedaegonPipelineV1PipelineSpecResourceGpuToJSON,
} from './IoGithub1eedaegonPipelineV1PipelineSpecResourceGpu';

/**
 * 
 * @export
 * @interface IoGithub1eedaegonPipelineV1PipelineSpecResource
 */
export interface IoGithub1eedaegonPipelineV1PipelineSpecResource {
    /**
     * 
     * @type {string}
     * @memberof IoGithub1eedaegonPipelineV1PipelineSpecResource
     */
    cpu?: string;
    /**
     * 
     * @type {IoGithub1eedaegonPipelineV1PipelineSpecResourceGpu}
     * @memberof IoGithub1eedaegonPipelineV1PipelineSpecResource
     */
    gpu?: IoGithub1eedaegonPipelineV1PipelineSpecResourceGpu;
    /**
     * 
     * @type {string}
     * @memberof IoGithub1eedaegonPipelineV1PipelineSpecResource
     */
    memory?: string;
}

/**
 * Check if a given object implements the IoGithub1eedaegonPipelineV1PipelineSpecResource interface.
 */
export function instanceOfIoGithub1eedaegonPipelineV1PipelineSpecResource(value: object): value is IoGithub1eedaegonPipelineV1PipelineSpecResource {
    return true;
}

export function IoGithub1eedaegonPipelineV1PipelineSpecResourceFromJSON(json: any): IoGithub1eedaegonPipelineV1PipelineSpecResource {
    return IoGithub1eedaegonPipelineV1PipelineSpecResourceFromJSONTyped(json, false);
}

export function IoGithub1eedaegonPipelineV1PipelineSpecResourceFromJSONTyped(json: any, ignoreDiscriminator: boolean): IoGithub1eedaegonPipelineV1PipelineSpecResource {
    if (json == null) {
        return json;
    }
    return {
        
        'cpu': json['cpu'] == null ? undefined : json['cpu'],
        'gpu': json['gpu'] == null ? undefined : IoGithub1eedaegonPipelineV1PipelineSpecResourceGpuFromJSON(json['gpu']),
        'memory': json['memory'] == null ? undefined : json['memory'],
    };
}

export function IoGithub1eedaegonPipelineV1PipelineSpecResourceToJSON(value?: IoGithub1eedaegonPipelineV1PipelineSpecResource | null): any {
    if (value == null) {
        return value;
    }
    return {
        
        'cpu': value['cpu'],
        'gpu': IoGithub1eedaegonPipelineV1PipelineSpecResourceGpuToJSON(value['gpu']),
        'memory': value['memory'],
    };
}

