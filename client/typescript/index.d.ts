import k8sClient from "@kubernetes/client-node";

declare namespace pipelineOperator {
  // General
  interface ObjectWith<ISpec, IStatus> extends k8sClient.KubernetesObject {
    spec?: ISpec;
    status?: IStatus;
  }

  // api/v1/task_types.go
  interface TaskSpec {
    name?: string;
    image?: string;
    command?: string;
    args?: string[];
  }
  interface TaskStatus {
    Jobs: number;
    createdDate: number;
    lastUpdatedDate: number;
  }
  interface Task extends ObjectWith<TaskSpec, TaskStatus> {}
  interface TaskList extends k8sClient.KubernetesListObject<Task> {}

  // api/v1/run_types.go
  namespace JobCategory {
    type JobCategory = PreRun | Run | PostRun;

    type PreRun = "preRun";
    type Run = "run";
    type PostRun = "postRun";

    type Map = {
      [state in JobState.JobState]: JobCategory;
    };
    enum map {
      initializing = "preRun",
      waiting = "preRun",
      stopping = "preRun",
      running = "run",
      deleting = "run",
      completed = "postRun",
      failed = "postRun",
      deleted = "postRun",
    }
  }

  namespace JobState {
    type JobState =
      | Init
      | Wait
      | Stop
      | Run
      | Deleting
      | Completed
      | Deleted
      | Failed;

    // PreRun
    type Init = "initializing";
    type Wait = "waiting";
    type Stop = "stopping";

    // Run
    type Run = "running";
    type Deleting = "deleting";

    // PostRun
    type Completed = "completed";
    type Deleted = "deleted";
    type Failed = "failed";

    type Order = {
      [state in JobState]: number;
    };
    enum stateOrder {
      "failed" = 8,
      "running" = 7,
      "initializing" = 6,
      "waiting" = 5,
      "completed" = 4,
      "deleting" = 3,
      "stopping" = 2,
      "deleted" = 1,
    }
  }

  interface RunJobState {
    name: string;
    jobState: JobState.JobState;
    reason: string;
  }

  namespace RunState {
    type RunState =
      | Init
      | Wait
      | Stop
      | Run
      | Deleting
      | Completed
      | Deleted
      | Failed;

    // PreRun
    type Init = "initializing";
    type Wait = "waiting";
    type Stop = "stopping";

    // Run
    type Run = "running";
    type Deleting = "deleting";

    // PostRun
    type Completed = "completed";
    type Deleted = "deleted";
    type Failed = "failed";
  }

  interface Job {
    name: string;
    namespace: string;
    image: string;
    command: string[];
    schedule: Schedule;
    resource: Resource;
  }
  interface RunSpec {
    schedule?: Schedule;
    volumes?: VolumeResource;
    trigger?: boolean;
    historyLimit?: HistoryLimit;
    jobs?: Job[];
    runBefore?: string[];
    inputs?: string[];
    outputs?: string[];
    resources?: Resource[];
    env?: {
      [k in string]: string;
    };
  }

  interface RunStatus {
    runState?: RunState;
    createdDate?: Date;
    lastUpdatedDate?: Date;
    jobStates?: RunJobState[];
    initializing?: number;
    waiting?: number;
    stopping?: number;
    running?: number;
    deleting?: number;
    completed?: number;
    deleted?: number;
    failed?: number;
  }

  interface Run extends ObjectWith<RunSpec, RunStatus> {}
  interface RunList extends k8sClient.KubernetesListObject<Run> {}

  // api/v1/pipeline_types.go
  interface HistoryLimit {
    amount: number;
    date: string;
  }

  interface Resource {
    cpu?: string;
    memory?: string;
    gpu?: GpuResource;
  }

  interface GpuResource {
    gpuType: string;
    amount: number;
  }

  interface VolumeResource {
    name: string;
    capacity: number;
    storage: string;
  }

  type ScheduleDate = string;

  interface Schedule {
    scheduleDate: ScheduleDate;
    endDate: string;
  }

  namespace ModeType {
    type modeType = auto | manual;

    // Default
    type auto = "auto";
    type manual = "manual";
  }

  interface PipelineTask {
    taskSpec?: TaskSpec;
    schedule?: Schedule;
    resource?: Resource;
    trigger?: boolean;
    runBefore?: string[];
    inputs?: string[];
    outputs?: string[];
    env?: {
      [k in string]: string;
    };
  }

  interface PipelineSpec {
    schedule?: Schedule;
    volumes?: VolumeResource[];
    trigger?: boolean;
    historyLimit?: HistoryLimit;
    tasks?: PipelineTask[];
    runBefore?: string[];
    inputs?: string[];
    outputs?: string[];
    resource?: Resource;
    env?: {
      [k in string]: string;
    };
  }
  interface PipelineStatus {
    Jobs: number;
    createdDate: number;
    lastUpdatedDate: number;
  }
  interface Pipeline extends ObjectWith<PipelineSpec, PipelineStatus> {}
}

export = pipelineOperator;
